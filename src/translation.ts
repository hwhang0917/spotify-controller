export const translation: Locale = {
  english: {
    adminLogin: {
      title: "Spotify Controller",
      intro: "Welcome to spotify-controller, please login to Spotify to begin.",
      requirementsTitle: "Requirements",
      requirements: [
        "Spotify account with premium subscription",
        "Custom application added to Spotify Developer Dashboard",
      ],
      loginButton: "Login to Spotify",
      unauthorized: "Unauthorized",
      gobackButton: "Go Back",
    },
    adminSettings: {
      title: "Settings",
      description: "Configure the settings for the Spotify Controller.",
      allowPausingSwitch: "Allow Pausing",
      allowPausingDescription: "Allow users to pause the current song.",
      allowSkippingSwitch: "Allow Skipping",
      allowSkippingDescription: "Allow users to skip the current song.",
      allowViewingSwitch: "Allow Viewing",
      allowViewingDescription: "Allow users to view the current playlist.",
      saveButton: "Save",
      exitButton: "Exit",
      toastMessage: "Settings saved successfully.",
    },
  },
  korean: {
    adminLogin: {
      title: "스포티파이 컨트롤러",
      intro:
        "스포티파이 컨트롤러에 오신 것을 환영합니다. 시작하려면 스포티파이에 로그인하세요.",
      requirementsTitle: "요구 사항",
      requirements: [
        "스포티파이 프리미엄 구독이 있는 계정",
        "스포티파이 개발자 대시보드에 사용자 지정 애플리케이션 추가",
      ],
      loginButton: "스포티파이에 로그인",
      unauthorized: "관리자만 접근할 수 있습니다.",
      gobackButton: "홈으로 가기",
    },
    adminSettings: {
      title: "설정",
      description: "스포티파이 컨트롤러의 설정을 구성합니다.",
      allowPausingSwitch: "일시 정지 허용",
      allowPausingDescription:
        "사용자가 현재 곡을 일시 정지할 수 있도록 허용합니다.",
      allowSkippingSwitch: "스킵 허용",
      allowSkippingDescription:
        "사용자가 현재 곡을 스킵할 수 있도록 허용합니다.",
      allowViewingSwitch: "보기 허용",
      allowViewingDescription:
        "사용자가 현재 플레이리스트를 볼 수 있도록 허용합니다.",
      saveButton: "저장",
      exitButton: "종료",
      toastMessage: "설정이 성공적으로 저장되었습니다.",
    },
  },
};
