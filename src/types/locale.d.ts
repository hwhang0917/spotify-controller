type Language = "english" | "korean";

interface LocalePerLang {
  adminLogin: {
    title: string;
    intro: string;
    requirementsTitle: string;
    requirements: string[];
    loginButton: string;
    unauthorized: string;
    gobackButton: string;
  };
  adminSettings: {
    title: string;
    description: string;
    allowPausingSwitch: string;
    allowPausingDescription: string;
    allowSkippingSwitch: string;
    allowSkippingDescription: string;
    allowViewingSwitch: string;
    allowViewingDescription: string;
    saveButton: string;
    exitButton: string;
    toastMessage: string;
  };
}

interface Locale {
  korean: LocalePerLang;
  english: LocalePerLang;
}
