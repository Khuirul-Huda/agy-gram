module.exports = {
  apps: [
    {
      name: "agy-gram",
      script: "./bin/agy-gram",
      cwd: "/home/huda/repo/agy-gram",
      instances: 1,
      exec_mode: "fork",
      autorestart: true,
      watch: false,
      max_memory_restart: "150M",
      restart_delay: 5000,
      env: {
        PATH: process.env.PATH,
        HOME: process.env.HOME,
      },
    },
  ],
};
