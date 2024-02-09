<p align="center">
   <img src="../public/icon.png" width="200px"/>
</p>

# Spotify Bot

> 공통 스포티파이 플레이리스트를 관리하는 웹 애플리케이션입니다.

[![Author](https://img.shields.io/badge/author-RunFridge-green?style=flat)](https://github.com/hwhang0917)
[![License](https://img.shields.io/github/license/RunFridge/film-book)](https://github.com/hwhang0917/spotify-bot/blob/master/LICENSE)

## 요구사항

- Node.js ^20.11.0
- 스포티파이 프리미엄 계정
- GUI 지원 브라우저

## 설치 가이드

1. 로컬 컴퓨터에 해당 레포지토리를 클론합니다.

2. `yarn` 커맨드를 사용하여 필요한 모듈을 설치합니다.

3. [스포티파이 대시보드](https://developer.spotify.com/dashboard/applications)로 이동하여 새로운 애플리케이션을 생성합니다.

4. Redirect URI에 해당 URI를 추가합니다: `http://localhost:3000/callback`.

5. 생성된 스포티파이 애플리케이션의 클라이언트 ID 그리고 클라이언트 시크릿을 복사합니다.

6. `.env.sample` 파일을 프로젝트 루트에 `.env`로 복사합니다.

7. 주어진 가이드에 따라 `.env` 파일에 필요한 정보를 입력합니다.

8. `yarn start` 커맨드로 서버를 실행합니다.

   - 서버는 기본적으로 시스템의 언어 설정을 따라갑니다. 한글로 고정하려면 아래 커맨드로 실행해야합니다.

   ```bash
   LANG=ko_KR.UTF-8 && yarn deploy
   ```

9. 브라우저가 열리면 설명을 참조합니다.

10. 공유기에서 할당된 내부 IP 주소의 `3000` 포트를 공유합니다 `http://${내부아이피}:3000` 또는 `3000` 포트를 외부에 공개합니다.
