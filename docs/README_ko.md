<p align="center">
   <img src="../public/icon.png" width="200px"/>
</p>

# Spotify Bot

> 공통 스포티파이 플레이리스트를 관리하는 슬랙 봇입니다.

[![Author](https://img.shields.io/badge/author-RunFridge-green?style=flat)](https://github.com/hwhang0917)
[![License](https://img.shields.io/github/license/RunFridge/film-book)](https://github.com/hwhang0917/spotify-bot/blob/master/LICENSE)

## 요구사항

- Node.js ^16.17.0
- 스포티파이 프리미엄 계정
- GUI 지원 브라우저

## 설치 가이드

1. 로컬 컴퓨터에 해당 레포지토리를 클론합니다.

2. `yarn` 커맨드를 사용하여 필요한 모듈을 설치합니다.

3. [스포티파이 대시보드](https://developer.spotify.com/dashboard/applications)로 이동하여 새로운 애플리케이션을 생성합니다.

4. Redirect URI에 해당 URI를 추가합니다: `http://localhost:5555/callback`.

5. 생성된 스포티파이 애플리케이션의 클라이언트 ID 그리고 클라이언트 시크릿을 복사합니다.

6. `.env.sample` 파일을 프로젝트 루트에 `.env`로 복사합니다.

7. 주어진 가이드에 따라 `.env` 파일에 필요한 정보를 입력합니다.

8. `yarn start:dev` 커맨드로 서버를 실행합니다.