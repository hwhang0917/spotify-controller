type Language = "english" | "korean";

interface LocalePerLang {
  unauthMain: {
    title: string;
    intro: string;
    requirementsTitle: string;
    requirements: string[];
    loginButton: string;
  };
}

interface Locale {
  korean: LocalePerLang;
  english: LocalePerLang;
}
