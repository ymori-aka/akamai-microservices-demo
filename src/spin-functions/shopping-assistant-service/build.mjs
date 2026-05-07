// build.mjs
import { build } from 'esbuild';
import path from 'path';
import { SpinEsbuildPlugin } from "@spinframework/build-tools/plugins/esbuild/index.js";
import fs from 'fs';

const spinPlugin = await SpinEsbuildPlugin();

let SourceMapPlugin = {
    name: 'excludeVendorFromSourceMap',
    setup(build) {
        build.onLoad({ filter: /node_modules\/.*\.(?:[cm]?js|tsx?)$/ }, args => {
            return {
                contents: fs.readFileSync(args.path, 'utf8')
                    + '\n//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbIiJdLCJtYXBwaW5ncyI6IkEifQ==',
                loader: 'default',
            }
        })
    },
}

await build({
    entryPoints: ['./src/index.ts'],
    outfile: './build/bundle.js',
    bundle: true,
    format: 'esm',
    platform: 'browser',
    sourcemap: true,
    minify: false,
    plugins: [spinPlugin, SourceMapPlugin],
    logLevel: 'error',
    loader: {
        '.ts': 'ts',
        '.tsx': 'tsx',
    },
    resolveExtensions: ['.ts', '.tsx', '.js'],
    sourceRoot: path.resolve(process.cwd(), 'src'),
});
