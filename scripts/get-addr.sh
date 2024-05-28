#!/usr/bin/env bash

SPOTIFY=$(cat << 'EOF'
        ⠀⠀⠀⠀⠀⠀⠀⢀⣠⣤⣤⣶⣶⣶⣶⣤⣤⣄⡀⠀⠀⠀⠀⠀⠀⠀
        ⠀⠀⠀⠀⢀⣤⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣤⡀⠀⠀⠀⠀
        ⠀⠀⠀⣴⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣦⠀⠀⠀
        ⠀⢀⣾⣿⡿⠿⠛⠛⠛⠉⠉⠉⠉⠛⠛⠛⠿⠿⣿⣿⣿⣿⣿⣷⡀⠀
        ⠀⣾⣿⣿⣇⠀⣀⣀⣠⣤⣤⣤⣤⣠⣀⣀⠀⠀⠀⠈⠙⠻⣿⣿⣷⠀
        ⢠⣿⣿⣿⣿⡿⠿⠟⠛⠛⠛⠛⠛⠘⠺⠿⢿⣿⣶⣤⣀⣠⣿⣿⣿⡄
        ⢸⣿⣿⣿⣿⣇⣀⣀⣤⣤⣤⣤⣤⣄⣀⣀⠀⠀⠉⠛⢿⣿⣿⣿⣿⡇
        ⠘⣿⣿⣿⣿⣿⠿⠿⠛⠛⠛⠛⠛⠛⠿⠿⣿⣶⣦⣤⣾⣿⣿⣿⣿⠃
        ⠀⢿⣿⣿⣿⣿⣤⣤⣤⣤⣶⣶⣦⣤⣤⣄⡀⠈⠙⣿⣿⣿⣿⣿⡿⠀
        ⠀⠈⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣾⣿⣿⣿⣿⡿⠁⠀
        ⠀⠀⠀⠻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟⠀⠀⠀
        ⠀⠀⠀⠀⠈⠛⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠛⠁⠀⠀⠀⠀
        ⠀⠀⠀⠀⠀⠀⠀⠈⠙⠛⠛⠿⠿⠿⠿⠛⠛⠋⠁⠀⠀⠀⠀⠀⠀⠀
EOF)

OUTPUT=""
LINE_BREAK="    ===============================    "

# Check if the os is MacOS
if [[ "$OSTYPE" == "darwin"* ]]; then
    OUTPUT=$(ifconfig | grep -Eo 'inet (addr:)?([0-9]*\.){3}[0-9]*' | grep -v '127.0.0.1' | awk '{ print "http://"$2":5555" }')
# Check if the os is Linux
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OUTPUT=$(ip addr | grep -Eo 'inet (addr:)?([0-9]*\.){3}[0-9]*' | grep 192 | awk '{ print "http://"$2":5555" }')
fi

# Print OUTPUT in the terminal as pink
echo -e "\033[32m$SPOTIFY\n    SHARE THE FOLLOWING LINK WITH YOUR FRIENDS \n $LINE_BREAK \n     $OUTPUT \n $LINE_BREAK\033[0m"
echo -e "\n\n\n"
