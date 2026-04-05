export type Locale = 'en' | 'fr';

export const translations = {
  en: {
    // Layout
    siteTitle: 'GoTK — LLM Output Proxy',
    siteDescription: 'GoTK — LLM Output Proxy. Reduce token usage by 48-98% by filtering command output before it reaches your LLM.',

    // Hero
    heroBadge: 'Go CLI Tool \u00B7 Open Source',
    heroTitle1: 'Cut LLM token usage',
    heroTitle2: 'by 87%',
    heroSubtitle: 'GoTK filters command output before it reaches your LLM. Fewer tokens, same information, faster responses.',
    heroCopy: 'Copy',
    heroCopied: 'Copied!',
    heroGithub: 'View on GitHub',

    // Before/After
    beforeAfterTitle: 'See the difference',
    beforeAfterSubtitle: 'Real command output — before and after GoTK',
    beforePrefix: 'Raw',
    afterPrefix: 'Filtered',
    lines: 'lines',

    // Features
    featuresTitle: 'Features',
    featuresSubtitle: 'Quality-first filtering \u2014 remove noise, preserve every error and warning',
    feature1Title: '18+ Command Filters',
    feature1Desc: 'Specialized filters for grep, git, go, docker, npm, terraform, kubectl, and more.',
    feature2Title: '6 Execution Modes',
    feature2Desc: 'Direct, pipe, exec, watch, daemon, and context search.',
    feature3Title: 'Secret Redaction',
    feature3Desc: 'API keys, tokens, passwords, and JWTs automatically replaced with [REDACTED].',
    feature4Title: 'Stack Trace Compression',
    feature4Desc: 'Go, Python, and Node.js traces condensed to cause + top frame.',
    feature5Title: 'Smart Truncation',
    feature5Desc: 'Head + tail preservation ensures errors and summaries are never cut.',
    feature6Title: 'Pattern Learning',
    feature6Desc: 'Teach GoTK project-specific noise patterns for custom filtering.',

    // Integrations
    integrationsTitle: 'Works with your AI tools',
    integrationsSubtitle: 'One-line setup for the most popular LLM coding assistants',
    claudeMethod: 'Native PreToolUse hook',
    aiderMethod: 'Shell proxy',
    cursorMethod: 'Terminal profile',
    continueMethod: 'Context provider',

    // Benchmarks
    benchTitle: 'Real-world benchmarks',
    benchSubtitle: 'Measured on real command output \u2014 verify yourself with',
    benchAvg: 'Average reduction:',
    benchAvgSuffix: 'across real-world workloads',

    // QuickStart
    quickStartTitle: 'Get started in 30 seconds',
    step1Title: 'Install',
    step2Title: 'Set up your tool',
    step2Note: 'Or see integrations for Aider, Cursor, Continue.dev',
    step3Title: 'Use it',
    step3Note: 'Or pipe: go test ./... | gotk --stats',

    // Footer
    footerDocs: 'Documentation',
    footerLicense: 'License (MIT)',
    footerBuiltWith: 'Built with Go. Site built with Astro.',

    // Language switcher
    langLabel: 'FR',
  },
  fr: {
    // Layout
    siteTitle: 'GoTK — Proxy de sortie LLM',
    siteDescription: 'GoTK — Proxy de sortie LLM. R\u00E9duisez la consommation de tokens de 48 \u00E0 98% en filtrant la sortie des commandes avant de l\u2019envoyer \u00E0 votre LLM.',

    // Hero
    heroBadge: 'Outil CLI Go \u00B7 Open Source',
    heroTitle1: 'R\u00E9duisez vos tokens LLM',
    heroTitle2: 'de 87%',
    heroSubtitle: 'GoTK filtre la sortie des commandes avant qu\u2019elle n\u2019atteigne votre LLM. Moins de tokens, m\u00EAme information, r\u00E9ponses plus rapides.',
    heroCopy: 'Copier',
    heroCopied: 'Copi\u00E9 !',
    heroGithub: 'Voir sur GitHub',

    // Before/After
    beforeAfterTitle: 'Voyez la diff\u00E9rence',
    beforeAfterSubtitle: 'Vraies sorties de commandes \u2014 avant et apr\u00E8s GoTK',
    beforePrefix: 'Brut',
    afterPrefix: 'Filtr\u00E9',
    lines: 'lignes',

    // Features
    featuresTitle: 'Fonctionnalit\u00E9s',
    featuresSubtitle: 'Filtrage qualit\u00E9 d\u2019abord \u2014 supprime le bruit, pr\u00E9serve chaque erreur et avertissement',
    feature1Title: '18+ filtres de commandes',
    feature1Desc: 'Filtres sp\u00E9cialis\u00E9s pour grep, git, go, docker, npm, terraform, kubectl, et plus.',
    feature2Title: '6 modes d\u2019ex\u00E9cution',
    feature2Desc: 'Direct, pipe, exec, watch, daemon, et recherche contextuelle.',
    feature3Title: 'Masquage de secrets',
    feature3Desc: 'Cl\u00E9s API, tokens, mots de passe et JWTs automatiquement remplac\u00E9s par [REDACTED].',
    feature4Title: 'Compression de stack traces',
    feature4Desc: 'Traces Go, Python et Node.js condens\u00E9es \u00E0 la cause + frame principale.',
    feature5Title: 'Troncature intelligente',
    feature5Desc: 'Pr\u00E9servation t\u00EAte + queue pour ne jamais couper erreurs et r\u00E9sum\u00E9s.',
    feature6Title: 'Apprentissage de patterns',
    feature6Desc: 'Enseignez \u00E0 GoTK les patterns de bruit sp\u00E9cifiques \u00E0 votre projet.',

    // Integrations
    integrationsTitle: 'Compatible avec vos outils IA',
    integrationsSubtitle: 'Configuration en une ligne pour les assistants de code LLM les plus populaires',
    claudeMethod: 'Hook PreToolUse natif',
    aiderMethod: 'Proxy shell',
    cursorMethod: 'Profil terminal',
    continueMethod: 'Fournisseur de contexte',

    // Benchmarks
    benchTitle: 'Benchmarks r\u00E9els',
    benchSubtitle: 'Mesur\u00E9 sur de vraies sorties de commandes \u2014 v\u00E9rifiez par vous-m\u00EAme avec',
    benchAvg: 'R\u00E9duction moyenne :',
    benchAvgSuffix: 'sur des charges de travail r\u00E9elles',

    // QuickStart
    quickStartTitle: 'D\u00E9marrez en 30 secondes',
    step1Title: 'Installer',
    step2Title: 'Configurer votre outil',
    step2Note: 'Ou voir les int\u00E9grations pour Aider, Cursor, Continue.dev',
    step3Title: 'Utiliser',
    step3Note: 'Ou en pipe : go test ./... | gotk --stats',

    // Footer
    footerDocs: 'Documentation',
    footerLicense: 'Licence (MIT)',
    footerBuiltWith: 'Construit avec Go. Site construit avec Astro.',

    // Language switcher
    langLabel: 'EN',
  },
} as const;

export type TranslationKey = keyof typeof translations.en;

export function t(locale: Locale, key: TranslationKey): string {
  return translations[locale][key];
}

export function getLocalePath(locale: Locale, path: string = ''): string {
  if (locale === 'en') return `/GoTK/${path}`;
  return `/GoTK/${locale}/${path}`;
}
