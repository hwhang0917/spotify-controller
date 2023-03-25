module.exports = {
  apps: [
    {
      name: "spotify-api",
      cwd: "./api",
      script: "yarn",
      args: "deploy",
    },
    {
      name: "spotify-app",
      cwd: "./app",
      script: "yarn",
      args: ["deploy", "-p", "5050"],
    },
  ],
  env: {
    NODE_ENV: "development",
  },
};
