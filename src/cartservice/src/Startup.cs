using System;
using System.Security.Cryptography.X509Certificates;
using System.Threading.Tasks;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Diagnostics.HealthChecks;
using Microsoft.AspNetCore.Hosting;
using Microsoft.AspNetCore.Http;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Diagnostics.HealthChecks;
using Microsoft.Extensions.Hosting;
using cartservice.cartstore;
using cartservice.services;
using Microsoft.Extensions.Caching.StackExchangeRedis;
using OpenTelemetry.Resources;
using OpenTelemetry.Trace;
using StackExchange.Redis;

namespace cartservice
{
    public class Startup
    {
        public Startup(IConfiguration configuration)
        {
            Configuration = configuration;
        }

        public IConfiguration Configuration { get; }

        // This method gets called by the runtime. Use this method to add services to the container.
        // For more information on how to configure your application, visit https://go.microsoft.com/fwlink/?LinkID=398940
        public void ConfigureServices(IServiceCollection services)
        {
            string redisAddress = Configuration["REDIS_ADDR"];
            // Path to a PEM-encoded CA certificate (mounted from a Secret).
            // Set this for Akamai Managed Valkey (Aiven), whose TLS chain is
            // issued by a per-account CA that isn't in the base image's trust
            // store — the runtime image is "chiseled" (no shell), so there's
            // no update-ca-certificates to lean on; we verify explicitly
            // in-process instead. Leave unset for the in-cluster Redis
            // Deployment (no TLS).
            string redisCaCertPath = Configuration["REDIS_CA_CERT_PATH"];
            string spannerProjectId = Configuration["SPANNER_PROJECT"];
            string spannerConnectionString = Configuration["SPANNER_CONNECTION_STRING"];
            string alloyDBConnectionString = Configuration["ALLOYDB_PRIMARY_IP"];

            if (!string.IsNullOrEmpty(redisAddress))
            {
                // REDIS_ADDR is a full StackExchange.Redis connection string
                // (e.g. "host:port,ssl=true,user=akmadmin,password=..." for
                // Managed Valkey, or just "host:port" for the in-cluster
                // Redis Deployment).
                var redisOptions = ConfigurationOptions.Parse(redisAddress);
                if (!string.IsNullOrEmpty(redisCaCertPath))
                {
                    var caCert = new X509Certificate2(redisCaCertPath);
                    redisOptions.CertificateValidation += (sender, certificate, chain, errors) =>
                    {
                        using var validationChain = new X509Chain();
                        validationChain.ChainPolicy.ExtraStore.Add(caCert);
                        validationChain.ChainPolicy.RevocationMode = X509RevocationMode.NoCheck;
                        validationChain.ChainPolicy.VerificationFlags = X509VerificationFlags.AllowUnknownCertificateAuthority;
                        return validationChain.Build(new X509Certificate2(certificate));
                    };
                }

                // Register a single IConnectionMultiplexer that BOTH the
                // distributed cache and OpenTelemetry's Redis instrumentation
                // can hook into. If we let AddStackExchangeRedisCache create
                // its own internal multiplexer we'd have no handle to attach
                // tracing to, so cart HMGET/HSET wouldn't produce spans.
                var muxer = ConnectionMultiplexer.Connect(redisOptions);
                services.AddSingleton<IConnectionMultiplexer>(muxer);
                services.AddStackExchangeRedisCache(options =>
                {
                    options.ConnectionMultiplexerFactory = () => Task.FromResult<IConnectionMultiplexer>(muxer);
                });
                services.AddSingleton<ICartStore, RedisCartStore>();
            }
            else if (!string.IsNullOrEmpty(spannerProjectId) || !string.IsNullOrEmpty(spannerConnectionString))
            {
                services.AddSingleton<ICartStore, SpannerCartStore>();
            }
            else if (!string.IsNullOrEmpty(alloyDBConnectionString))
            {
                Console.WriteLine("Creating AlloyDB cart store");
                services.AddSingleton<ICartStore, AlloyDBCartStore>();
            }
            else
            {
                Console.WriteLine("Redis cache host(hostname+port) was not specified. Starting a cart service using in memory store");
                services.AddDistributedMemoryCache();
                services.AddSingleton<ICartStore, RedisCartStore>();
            }

            // OpenTelemetry — server / outbound-gRPC / Redis client spans
            // exported via OTLP to the otel-collector (env-driven endpoint
            // OTEL_EXPORTER_OTLP_ENDPOINT is read by the OTLP exporter).
            services.AddOpenTelemetry()
                .ConfigureResource(r => r.AddService("cartservice"))
                .WithTracing(builder =>
                {
                    builder
                        .AddAspNetCoreInstrumentation()
                        .AddGrpcClientInstrumentation()
                        .AddRedisInstrumentation()
                        .AddOtlpExporter();
                });

            services.AddGrpc();
        }

        // This method gets called by the runtime. Use this method to configure the HTTP request pipeline.
        public void Configure(IApplicationBuilder app, IWebHostEnvironment env)
        {
            if (env.IsDevelopment())
            {
                app.UseDeveloperExceptionPage();
            }

            app.UseRouting();

            app.UseEndpoints(endpoints =>
            {
                endpoints.MapGrpcService<CartService>();
                endpoints.MapGrpcService<cartservice.services.HealthCheckService>();

                endpoints.MapGet("/", async context =>
                {
                    await context.Response.WriteAsync("Communication with gRPC endpoints must be made through a gRPC client. To learn how to create a client, visit: https://go.microsoft.com/fwlink/?linkid=2086909");
                });
            });
        }
    }
}
