FROM node:16-alpine

WORKDIR site
RUN wget https://js-dos.com/6.22/current/js-dos.js && \ 
    wget https://js-dos.com/6.22/current/wdosbox.js && \    
    wget https://js-dos.com/6.22/current/wdosbox.wasm.js
RUN npm install -g serve


ARG GAME_URL
RUN wget -O game.zip $GAME_URL

ARG GAME_ARGS
COPY index.html .
RUN sed -i s/GAME_ARGS/$GAME_ARGS/ index.html

ENTRYPOINT npx serve -l tcp://0.0.0.0:8000