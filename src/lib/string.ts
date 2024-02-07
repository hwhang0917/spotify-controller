export const getListFormatter = (language: Language) => {
  return new Intl.ListFormat(language === "korean" ? "ko-KR" : "en-US");
};
