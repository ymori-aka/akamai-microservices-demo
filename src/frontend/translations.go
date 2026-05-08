package main

// jaProduct holds Japanese translations for a product
type jaProduct struct {
	Name        string
	Description string
}

// jaTranslations maps product ID to Japanese name and description
var jaTranslations = map[string]jaProduct{
	"AKMT001": {
		Name:        "Akamai ユニセックス 半袖Tシャツ - ブルー",
		Description: "Akamaiの「We Move Mountains」クルーネックTシャツをブルーでスタイリッシュに着こなそう。柔らかくて通気性に優れたコットンブレンド素材で、毎日の限界に挑む人のための力強いTシャツです。",
	},
	"AKMT002": {
		Name:        "Akamai ユニセックス 半袖Tシャツ - グレー",
		Description: "Akamaiの「We Move Mountains」Tシャツをヘザーグレーで。象徴的なマウンテングラフィックのソフトコットンブレンド素材。カジュアルな日常や技術系イベント、週末のアドベンチャーに最適です。",
	},
	"AKMT003": {
		Name:        "Akamai ウィメンズ Vネック Tシャツ",
		Description: "Akamaiの「We Move Mountains」Tシャツをブルーのウィメンズ Vネックカットで。軽量でソフトなコットン素材にリラックスフィット。Akamaiスピリットをスタイリッシュに表現しましょう。",
	},
	"AKMT004": {
		Name:        "Akamai ユニセックス 半袖Tシャツ - ネイビー",
		Description: "Akamaiのサーフ/ウェーブアートグラフィックがあしらわれたスタイリッシュなネイビーTシャツ。エッジの力を表現したダイナミックなデザイン。柔らかくて通気性抜群のコットン素材でどんな場面にもぴったりです。",
	},
	"AKMT005": {
		Name:        "Akamai メンズ アディダス ポロシャツ - ネイビー",
		Description: "Akamai × Adidas のプレミアムポロシャツ、ネイビーヘザーカラー。吸湿速乾クライマライト素材にAkamaiロゴをスリーブに刺繍。プロフェッショナルで快適、パフォーマンス重視の一枚です。",
	},
	"AKMT006": {
		Name:        "Akamai ウィメンズ アディダス ポロシャツ - ネイビー",
		Description: "Akamai × Adidas ポロシャツのウィメンズテーラードフィット、ネイビーヘザーカラー。Adidas の吸湿速乾テクノロジーとスリーブのAkamaiロゴが特徴。技術系イベントであなたのスタイルを格上げします。",
	},
	"AKMT007": {
		Name:        "Akamai メンズ ベスト - ブラック/グレー",
		Description: "Akamaiロゴ入りプレミアム保温ベスト。ブラックの上部にグレーのキルティング下部、オレンジのアクセントジッパー。バルキーにならない軽量の暖かさでオフィスやアウトドアのレイヤリングに最適です。",
	},
	"AKMT008": {
		Name:        "Akamai ウィメンズ フリースジャケット - ネイビー",
		Description: "Akamaiロゴ入りネイビーのフルジップフリースジャケット。スタンドアップカラー、ジップポケット、オレンジのアクセントジッパーが特徴。データセンターからアウトドアまで、どこでも暖かく過ごせます。",
	},
	"AKMT009": {
		Name:        "Akamai メンズ フリースジャケット - ネイビー",
		Description: "Akamaiロゴ入りネイビーのプレミアムフルジップフリースジャケット。オレンジのアクセントジッパー、胸のジップポケット、快適なアスレチックフィット。どんなアドベンチャーにも対応する頼れるアウターです。",
	},
	"AKMT010": {
		Name:        "Akamai プルオーバーパーカー - ネイビー",
		Description: "胸元にAkamaiロゴを刺繍したクラシックなネイビーのプルオーバーパーカー。カンガルーポケットと調節可能なドローストリングフード付き。柔らかくて心地よく、毎日のウェアに最適です。",
	},
	"AKMT011": {
		Name:        "Akamai フルジップパーカー - チャコール",
		Description: "Akamaiロゴ入りのスタイリッシュなチャコールのフルジップパーカー。カンガルーポケットとホワイトのドローストリングアクセント付き。軽量ながら暖かく、技術系プロフェッショナルのための理想的なカジュアルレイヤーです。",
	},
	"AKMT012": {
		Name:        "Akamai テックソックス",
		Description: "頭のてっぺんから足のつま先まで Akamai プライドを表現！クラウド、サーバー、ロボットなどのカラフルなテックアイコンが描かれたネイビーのソックス。テック愛好家必携のアイテムです。",
	},
	"AKMT013": {
		Name:        "Akamai × ザ・ノース・フェイス トラッカーキャップ",
		Description: "Akamai と The North Face のプレミアムコラボ。構造的なブラックフロントにホワイトメッシュバックのスナップバックキャップで、Akamaiロゴを前面に刺繍。エッジエクスプローラーのために作られた一品です。",
	},
	"AKMT014": {
		Name:        "Akamai ビーチタオル - ブルー（Dock & Bay）",
		Description: "Akamai × Dock & Bay のプレミアム速乾ビーチタオル、ブルーとホワイトのストライプ柄。100%リサイクル素材製で、通常のタオルの3倍速く乾きます。ビーチ、ジム、旅行に最適な一枚です。",
	},
	"AKMT015": {
		Name:        "Akamai ビーチタオル - オレンジ（Dock & Bay）",
		Description: "Akamai × Dock & Bay のプレミアム速乾ビーチタオル、オレンジとホワイトのストライプ柄。100%リサイクル素材製。コンパクトで軽量、吸水性も抜群。どんなアドベンチャーにも対応します。",
	},
	"AKMT016": {
		Name:        "Akamai キャンバストートバッグ",
		Description: "Akamaiロゴ入りネイビーアクセントのクラシックなクリームキャンバストートバッグ。頑丈に補強されたハンドルと広々としたメインコンパートメント。エコフレンドリーで日常使いにスタイリッシュな一品です。",
	},
	"AKMT017": {
		Name:        "Akamai 40oz タンブラー（ハンドル付き）",
		Description: "スレートブルーのAkamaiロゴ入り巨大40ozハンドル＆ストロー付き保温タンブラー。24時間以上飲み物を冷たく保ちます。長時間のコーディングセッションに最高の水分補給パートナーです。",
	},
	"AKMT018": {
		Name:        "Akamai ノートブック",
		Description: "カバーにAkamaiロゴをデボス加工したプレミアムネイビーハードカバーノートブック。ゴムバンドクロージャー、リボンしおり、160ページの罫線入りページ付き。どんなカンファレンスでもアイデアを書き留めるのに最適です。",
	},
	"AKMT019": {
		Name:        "Akamai スタイラスペン",
		Description: "Akamaiロゴ入りのスリムなシルバースタイラスペン。タッチスクリーン対応の精密ペン先とボールペンを両端に搭載。現代のテックプロフェッショナルにとって最高のツールです。",
	},
	"AKMT020": {
		Name:        "Akamai ポップソケット",
		Description: "Akamaiロゴ入りのスリークなブラックポップソケット。スマートフォンやケースの背面に取り付けて、安定したグリップ、ハンズフリービューのスタンド、ケーブル管理システムとして機能します。日常の必需品です。",
	},
	"AKMT021": {
		Name:        "Akamai ゴルフボール",
		Description: "スタイリッシュなネイビーのギフトボックス入りAkamai ブランドのプレミアムゴルフボール。Akamaiロゴとゴルファーのシルエットがデザインされています。ゴルフ好きなテックプロフェッショナルへの完璧なギフト。3球セット。",
	},
	"AKMT022": {
		Name:        "Akamai スマートフォンスタンド＆リングライト",
		Description: "Akamaiブランドのスマートフォンスタンドとクリップオンリングライトのコンボでビデオ会議を格上げしましょう。調節可能な明るさでどんな環境でも完璧な照明を実現。コンパクトでポータブルなデザインです。",
	},
	"AKMT023": {
		Name:        "Akamai 25周年記念 クーラーバッグ",
		Description: "「25 Years of One Akamai」を記念したプレミアム保温クーラーバッグ（ブラック）。ブルーとオレンジの25周年記念ロゴをデザイン。6缶以上収納可能でショルダーストラップ付きです。",
	},
	"AKMT024": {
		Name:        "Akamai 25周年記念 タンブラー",
		Description: "「25 Years of One Akamai」を記念したスリークなマットブラック保温タンブラー。記念ロゴをレーザー彫刻。二重壁真空断熱で飲み物を最適な温度に保ちます。",
	},
	"AKMT025": {
		Name:        "Akamai 25周年記念 キャップ",
		Description: "「25 Years of One Akamai」を祝う25周年記念ベースボールキャップ。フロントにカラフルな記念ロゴを刺繍したブラックの構造的なキャップで、オレンジのブリムライニングが特徴。調節可能で完璧なフィット感を実現します。",
	},
	"AKMT026": {
		Name:        "Akamai ラップトップバックパック",
		Description: "オレンジのアクセントジッパーが特徴のスリークなグレーとブラックのAkamaiラップトップバックパック。パッド入りラップトップコンパートメント、フロントジップポケット、トップハンドル付き。移動中のテックプロフェッショナルのために設計されています。",
	},
	"AKMT027": {
		Name:        "Akamai ギフトカード - $50",
		Description: "Akamaiスワッグをプレゼントしよう！このギフトカードはAkamaiストアの全商品に使用できます。チームメンバー、顧客、エッジを愛するテック愛好家へのパーフェクトギフトです。",
	},
	"AKMT028": {
		Name:        "PEACE FOR ALL Tシャツ/アカマイ　白",
		Description: "25年以上前、Akamaiは今日のインターネットが自在に使える時代の礎を築きました。このTシャツのデザインは、そんなインターネット黎明期へのオマージュです。カラーは、当時「ベージュの箱」と呼ばれたコンピューターのプラスチック製の筐体をイメージしました。胸もとのハートは、インターネットが人々をつなぎ、より良い世界へ導いてきたことを表現しています。Tシャツの背面にプリントされているのは本物のコードです。それは、インターネットを支えるオープンソースのしくみを象徴する基本ソフト Linux にも通じます。世界中で愛されるこれらの共通言語の力を借りて、Akamaiが、世界を導くブランドとその先のユーザーとを結びつけ、より安全で、よりつながる未来を共に築いていきたい ―― そんな想いを、この一枚に込めました。",
	},
	"AKMT029": {
		Name:        "PEACE FOR ALL Tシャツ/アカマイ　黒",
		Description: "毎日、何十億もの人々が、オンラインで仕事、遊び、学び、ショッピング、アイデアの共有をしながら、つながっています。このデザインに使われている「texture」というコードは、デジタル体験の裏側にある共通言語を表現しています。Akamaiは 25 年にわたり、世界中の人々の人生をより豊かなものにするために、快適で安心なインターネット環境を築いてきました。これからも、世界をより安全に、より深く結びつける一役を担いたいと願っています。",
	},
}

// koProduct holds Korean translations for a product
type koProduct struct {
	Name        string
	Description string
}

// koTranslations maps product ID to Korean name and description
var koTranslations = map[string]koProduct{
	"AKMT001": {
		Name:        "Akamai 유니섹스 반소매 티셔츠 - 블루",
		Description: "파란색의 Akamai 'We Move Mountains' 크루넥 티셔츠. 부드럽고 통기성 좋은 코튼 블렌드 소재로, 매일 한계에 도전하는 이들을 위한 강인한 티셔츠입니다.",
	},
	"AKMT002": {
		Name:        "Akamai 유니섹스 반소매 티셔츠 - 그레이",
		Description: "헤더 그레이 색상의 Akamai 'We Move Mountains' 티셔츠. 상징적인 마운틴 그래픽의 소프트 코튼 블렌드. 일상, 기술 행사, 주말 어드벤처에 완벽합니다.",
	},
	"AKMT003": {
		Name:        "Akamai 여성용 V넥 티셔츠",
		Description: "블루 여성용 V넥 컷의 Akamai 'We Move Mountains' 티셔츠. 가볍고 부드러운 코튼 소재에 릴렉스 핏. Akamai 스피릿을 스타일리시하게 표현하세요.",
	},
	"AKMT004": {
		Name:        "Akamai 유니섹스 반소매 티셔츠 - 네이비",
		Description: "서핑/웨이브 아트 그래픽이 프린트된 스타일리시한 네이비 티셔츠. 엣지의 힘을 표현한 다이나믹한 디자인. 부드럽고 통기성 좋은 코튼 소재로 어디서든 활약합니다.",
	},
	"AKMT005": {
		Name:        "Akamai 남성용 아디다스 폴로셔츠 - 네이비",
		Description: "Akamai × Adidas 프리미엄 폴로셔츠, 네이비 헤더 컬러. 흡습속건 클라이마라이트 소재에 소매에 Akamai 로고 자수. 프로페셔널하고 편안한 퍼포먼스 아이템.",
	},
	"AKMT006": {
		Name:        "Akamai 여성용 아디다스 폴로셔츠 - 네이비",
		Description: "Akamai × Adidas 폴로셔츠 여성용 테일러드 핏, 네이비 헤더. Adidas 흡습속건 기술과 소매 Akamai 로고 자수. 기술 행사에서 스타일을 한층 높여보세요.",
	},
	"AKMT007": {
		Name:        "Akamai 남성용 베스트 - 블랙/그레이",
		Description: "Akamai 로고가 새겨진 프리미엄 보온 베스트. 블랙 상부와 그레이 퀼팅 하부, 오렌지 액센트 지퍼. 가볍고 따뜻해 오피스와 아웃도어 레이어링에 최적.",
	},
	"AKMT008": {
		Name:        "Akamai 여성용 플리스 재킷 - 네이비",
		Description: "Akamai 로고 자수 네이비 풀집 플리스 재킷. 스탠드업 칼라, 집 포켓, 오렌지 액센트 지퍼. 데이터센터부터 야외까지 어디서나 따뜻하게.",
	},
	"AKMT009": {
		Name:        "Akamai 남성용 플리스 재킷 - 네이비",
		Description: "Akamai 로고 자수 네이비 프리미엄 풀집 플리스 재킷. 오렌지 액센트 지퍼, 가슴 집 포켓, 편안한 애슬레틱 핏. 어떤 모험에도 믿음직한 아우터.",
	},
	"AKMT010": {
		Name:        "Akamai 풀오버 후디 - 네이비",
		Description: "가슴에 Akamai 로고를 자수한 클래식 네이비 풀오버 후디. 캥거루 포켓과 조절 가능한 드로스트링 후드 포함. 부드럽고 편안해 매일 입기 좋습니다.",
	},
	"AKMT011": {
		Name:        "Akamai 풀집 후디 - 차콜",
		Description: "Akamai 로고 자수의 스타일리시한 차콜 풀집 후디. 캥거루 포켓과 화이트 드로스트링 액센트 포함. 가볍고 따뜻해 기술 전문가에게 이상적인 캐주얼 레이어.",
	},
	"AKMT012": {
		Name:        "Akamai 테크 양말",
		Description: "발끝까지 Akamai 자부심을! 클라우드, 서버, 로봇 등 컬러풀한 테크 아이콘이 그려진 네이비 양말. 테크 애호가 필수 아이템.",
	},
	"AKMT013": {
		Name:        "Akamai × 더 노스 페이스 트래커 캡",
		Description: "Akamai와 The North Face의 프리미엄 콜라보. 구조적인 블랙 프론트에 화이트 메시 백의 스냅백 캡, 전면 Akamai 로고 자수. 엣지 탐험가를 위한 아이템.",
	},
	"AKMT014": {
		Name:        "Akamai 비치 타올 - 블루 (Dock & Bay)",
		Description: "Akamai × Dock & Bay 프리미엄 속건 비치 타올, 블루와 화이트 스트라이프. 100% 재활용 소재로 제작, 일반 타올보다 3배 빨리 건조. 해변, 헬스장, 여행에 완벽.",
	},
	"AKMT015": {
		Name:        "Akamai 비치 타올 - 오렌지 (Dock & Bay)",
		Description: "Akamai × Dock & Bay 프리미엄 속건 비치 타올, 오렌지와 화이트 스트라이프. 100% 재활용 소재. 컴팩트하고 가벼우며 흡수력도 탁월. 모든 모험에 대응.",
	},
	"AKMT016": {
		Name:        "Akamai 캔버스 토트백",
		Description: "Akamai 로고 자수의 네이비 액센트 클래식 크림 캔버스 토트백. 튼튼한 핸들과 넉넉한 메인 공간. 친환경적이며 일상 사용에 스타일리시한 아이템.",
	},
	"AKMT017": {
		Name:        "Akamai 40oz 텀블러 (핸들 포함)",
		Description: "Akamai 로고의 슬레이트 블루 40oz 핸들 & 스트로우 보온 텀블러. 24시간 이상 음료를 차갑게 유지. 긴 코딩 세션을 위한 최고의 수분 보충 파트너.",
	},
	"AKMT018": {
		Name:        "Akamai 노트북",
		Description: "Akamai 로고 데보스 가공의 프리미엄 네이비 하드커버 노트북. 고무밴드 클로저, 리본 책갈피, 160페이지 줄이 있는 페이지. 어느 컨퍼런스에서도 아이디어를 기록하기에 최적.",
	},
	"AKMT019": {
		Name:        "Akamai 스타일러스 펜",
		Description: "Akamai 로고가 새겨진 슬림 실버 스타일러스 펜. 터치스크린 호환 정밀 펜촉과 볼펜을 양단에 탑재. 현대 기술 전문가를 위한 최고의 도구.",
	},
	"AKMT020": {
		Name:        "Akamai 팝 소켓",
		Description: "Akamai 로고의 세련된 블랙 팝 소켓. 스마트폰이나 케이스 뒷면에 부착해 안정적인 그립, 핸즈프리 스탠드, 케이블 관리 시스템으로 활용. 일상 필수품.",
	},
	"AKMT021": {
		Name:        "Akamai 골프공",
		Description: "스타일리시한 네이비 선물 박스 포장의 Akamai 브랜드 프리미엄 골프공. Akamai 로고와 골퍼 실루엣 디자인. 골프 좋아하는 기술 전문가에게 완벽한 선물. 3개 세트.",
	},
	"AKMT022": {
		Name:        "Akamai 스마트폰 스탠드 & 링 라이트",
		Description: "Akamai 브랜드 스마트폰 스탠드와 클립온 링 라이트 콤보로 화상회의를 업그레이드하세요. 조절 가능한 밝기로 어떤 환경에서도 완벽한 조명 구현. 컴팩트하고 포터블한 디자인.",
	},
	"AKMT023": {
		Name:        "Akamai 25주년 기념 쿨러백",
		Description: "'25 Years of One Akamai'를 기념한 프리미엄 보온 쿨러백 (블랙). 블루와 오렌지의 25주년 기념 로고 디자인. 6캔 이상 수납 가능, 숄더 스트랩 포함.",
	},
	"AKMT024": {
		Name:        "Akamai 25주년 기념 텀블러",
		Description: "'25 Years of One Akamai'를 기념한 세련된 매트 블랙 보온 텀블러. 기념 로고 레이저 각인. 이중벽 진공 단열로 음료를 최적 온도로 유지.",
	},
	"AKMT025": {
		Name:        "Akamai 25주년 기념 캡",
		Description: "'25 Years of One Akamai'를 기념하는 25주년 기념 베이스볼 캡. 전면에 컬러풀한 기념 로고 자수의 블랙 구조적 캡, 오렌지 브림 라이닝. 조절 가능해 완벽한 핏 실현.",
	},
	"AKMT026": {
		Name:        "Akamai 랩톱 백팩",
		Description: "오렌지 액센트 지퍼가 특징인 세련된 그레이와 블랙 Akamai 랩톱 백팩. 패딩 처리된 랩톱 수납공간, 전면 집 포켓, 탑 핸들 포함. 이동 중인 기술 전문가를 위해 설계.",
	},
	"AKMT027": {
		Name:        "Akamai 기프트 카드 - $50",
		Description: "Akamai 스웨그를 선물하세요! 이 기프트 카드는 Akamai 스토어의 모든 상품에 사용 가능합니다. 팀원, 고객, 엣지를 사랑하는 기술 애호가에게 완벽한 선물.",
	},
	"AKMT028": {
		Name:        "PEACE FOR ALL 티셔츠/아카마이 화이트",
		Description: "25년 이상 전, Akamai는 오늘날 인터넷이 자유롭게 사용되는 시대의 토대를 마련했습니다. 이 티셔츠 디자인은 인터넷 초창기에 대한 오마주입니다. 컬러는 당시 '베이지 박스'라 불리던 컴퓨터 플라스틱 케이스를 연상시킵니다. 가슴의 하트는 인터넷이 사람들을 연결하고 더 나은 세상으로 이끌어왔음을 표현합니다.",
	},
	"AKMT029": {
		Name:        "PEACE FOR ALL 티셔츠/아카마이 블랙",
		Description: "매일 수십억 명의 사람들이 온라인으로 일하고, 놀고, 배우고, 쇼핑하며 아이디어를 나눕니다. Akamai는 25년간 전 세계 사람들의 삶을 더 풍요롭게 만들기 위해 안전하고 편안한 인터넷 환경을 구축해왔습니다.",
	},
}

// zhProduct holds Chinese (Simplified) translations for a product
type zhProduct struct {
	Name        string
	Description string
}

// zhTranslations maps product ID to Chinese (Simplified) name and description
var zhTranslations = map[string]zhProduct{
	"AKMT001": {
		Name:        "Akamai 中性款短袖T恤 - 蓝色",
		Description: "以蓝色演绎Akamai「We Move Mountains」圆领T恤。采用柔软透气的棉混纺面料，专为每天挑战极限的人打造，充满力量感。",
	},
	"AKMT002": {
		Name:        "Akamai 中性款短袖T恤 - 灰色",
		Description: "灰色系Akamai「We Move Mountains」T恤，饰以标志性山峰图案，采用柔软棉混纺面料。适合日常休闲、科技活动及周末冒险。",
	},
	"AKMT003": {
		Name:        "Akamai 女款V领T恤",
		Description: "蓝色女款V领剪裁的Akamai「We Move Mountains」T恤。轻盈柔软的棉质面料，宽松版型。以时尚方式展现Akamai精神。",
	},
	"AKMT004": {
		Name:        "Akamai 中性款短袖T恤 - 藏青色",
		Description: "印有冲浪/波浪艺术图案的时尚藏青色T恤，动感设计彰显边缘力量。柔软透气的棉质面料，适合各种场合。",
	},
	"AKMT005": {
		Name:        "Akamai 男款阿迪达斯Polo衫 - 藏青色",
		Description: "Akamai × Adidas高端Polo衫，藏青色混色。采用吸湿排汗Climalite面料，袖部绣有Akamai标志。专业舒适，性能卓越。",
	},
	"AKMT006": {
		Name:        "Akamai 女款阿迪达斯Polo衫 - 藏青色",
		Description: "Akamai × Adidas女款修身Polo衫，藏青色混色。Adidas吸湿排汗科技与袖部Akamai刺绣徽标。提升您在科技活动中的时尚品位。",
	},
	"AKMT007": {
		Name:        "Akamai 男款背心 - 黑/灰",
		Description: "印有Akamai标志的高端保暖背心。黑色上部搭配灰色绗缝下部，橙色点缀拉链。轻盈保暖，是办公室和户外叠穿的理想选择。",
	},
	"AKMT008": {
		Name:        "Akamai 女款摇粒绒夹克 - 藏青色",
		Description: "绣有Akamai标志的藏青色全拉链摇粒绒夹克。立领设计，拉链口袋，橙色点缀拉链。从数据中心到户外，随处保暖。",
	},
	"AKMT009": {
		Name:        "Akamai 男款摇粒绒夹克 - 藏青色",
		Description: "绣有Akamai标志的藏青色高端全拉链摇粒绒夹克。橙色点缀拉链，胸部拉链口袋，舒适运动版型。是任何冒险的可靠外套。",
	},
	"AKMT010": {
		Name:        "Akamai 套头卫衣 - 藏青色",
		Description: "胸前绣有Akamai标志的经典藏青色套头卫衣。附袋鼠口袋和可调节抽绳帽。柔软舒适，适合日常穿着。",
	},
	"AKMT011": {
		Name:        "Akamai 拉链卫衣 - 炭灰色",
		Description: "绣有Akamai标志的时尚炭灰色全拉链卫衣。附袋鼠口袋和白色抽绳点缀。轻盈保暖，是科技专业人士的理想休闲外套。",
	},
	"AKMT012": {
		Name:        "Akamai 科技袜",
		Description: "从头到脚展现Akamai自豪感！印有云朵、服务器、机器人等彩色科技图标的藏青色袜子。科技爱好者必备单品。",
	},
	"AKMT013": {
		Name:        "Akamai × The North Face 追踪者棒球帽",
		Description: "Akamai与The North Face的高端联名款。黑色硬挺帽身搭配白色网眼帽背的快扣棒球帽，前方绣有Akamai标志。专为边缘探索者打造。",
	},
	"AKMT014": {
		Name:        "Akamai 沙滩巾 - 蓝色（Dock & Bay）",
		Description: "Akamai × Dock & Bay高端速干沙滩巾，蓝白条纹设计。100%回收材料制成，干燥速度是普通毛巾的3倍。适合海滩、健身房及旅行。",
	},
	"AKMT015": {
		Name:        "Akamai 沙滩巾 - 橙色（Dock & Bay）",
		Description: "Akamai × Dock & Bay高端速干沙滩巾，橙白条纹设计。100%回收材料制成，紧凑轻便，吸水性出色。应对各种冒险。",
	},
	"AKMT016": {
		Name:        "Akamai 帆布托特包",
		Description: "藏青色点缀绣有Akamai标志的经典奶白色帆布托特包。加固提手和宽敞主袋。环保且时尚，适合日常使用。",
	},
	"AKMT017": {
		Name:        "Akamai 40oz 保温杯（带手柄）",
		Description: "板岩蓝配Akamai标志的40oz巨型带手柄和吸管保温杯。保冷超过24小时。长时间编程会话的最佳补水伙伴。",
	},
	"AKMT018": {
		Name:        "Akamai 笔记本",
		Description: "封面压印Akamai标志的高端藏青色硬封面笔记本。附橡皮筋封口、丝带书签和160页横线页面。是任何会议记录想法的绝佳选择。",
	},
	"AKMT019": {
		Name:        "Akamai 触控笔",
		Description: "印有Akamai标志的纤细银色触控笔。两端分别配有触屏兼容精密笔尖和圆珠笔。是现代科技专业人士的最佳工具。",
	},
	"AKMT020": {
		Name:        "Akamai 手机支架扣（PopSocket）",
		Description: "印有Akamai标志的简洁黑色PopSocket手机支架扣。贴于手机或手机壳背面，可用作稳固握持、免提支架和线缆管理。日常必备品。",
	},
	"AKMT021": {
		Name:        "Akamai 高尔夫球",
		Description: "配有时尚藏青色礼盒的Akamai品牌高端高尔夫球，印有Akamai标志和高尔夫球手剪影设计。是送给热爱高尔夫的科技专业人士的完美礼物。3球一组。",
	},
	"AKMT022": {
		Name:        "Akamai 手机支架与补光灯",
		Description: "用Akamai品牌手机支架和夹式补光灯组合提升视频会议体验。亮度可调，任何环境下均可实现完美照明。设计紧凑便携。",
	},
	"AKMT023": {
		Name:        "Akamai 25周年纪念保温袋",
		Description: "纪念「25 Years of One Akamai」的高端保温袋（黑色）。饰有蓝橙25周年纪念标志。可装6罐以上，附肩带。",
	},
	"AKMT024": {
		Name:        "Akamai 25周年纪念保温杯",
		Description: "纪念「25 Years of One Akamai」的简约哑光黑保温杯，激光雕刻纪念标志。双层真空隔热，饮品温度持久保持。",
	},
	"AKMT025": {
		Name:        "Akamai 25周年纪念棒球帽",
		Description: "庆祝「25 Years of One Akamai」的25周年纪念棒球帽。黑色硬挺帽身前方绣有彩色纪念标志，橙色帽檐内衬。可调节尺寸，完美贴合。",
	},
	"AKMT026": {
		Name:        "Akamai 笔记本电脑双肩包",
		Description: "以橙色点缀拉链为特色的时尚灰黑双色Akamai笔记本电脑双肩包。附加垫笔记本隔层、前置拉链口袋和顶部提手。专为移动科技专业人士设计。",
	},
	"AKMT027": {
		Name:        "Akamai 礼品卡 - $50",
		Description: "赠送Akamai周边礼品！此礼品卡可在Akamai商店所有商品中使用。是送给团队成员、客户及热爱边缘技术爱好者的完美礼物。",
	},
	"AKMT028": {
		Name:        "PEACE FOR ALL T恤/Akamai 白色",
		Description: "25年前，Akamai为当今互联网的自由使用奠定了基础。这件T恤的设计是对互联网初创时代的致敬。颜色灵感来源于当时被称为「米色盒子」的电脑塑料外壳。胸前的爱心象征着互联网将人们相连、引领世界走向更美好未来的寓意。",
	},
	"AKMT029": {
		Name:        "PEACE FOR ALL T恤/Akamai 黑色",
		Description: "每天，数十亿人在网上工作、娱乐、学习、购物和分享创意，彼此相连。Akamai 25年来为全球用户构筑安全舒适的网络环境，致力于让每个人的生活更加丰富多彩。",
	},
}
