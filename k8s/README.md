# Эксплуатация и разработка в Kubernetes
## Урок 01
Команда для сборки докер образа


```docker build --platform="linux/amd64" --build-arg GAME_TAG=dyna --build-arg GAME_URL=https://code.s3.yandex.net/Kubernetes/dyna.zip --build-arg GAME_ARGS=dyna.exe -t mycool:bomberman .```


Проверить, что образ создался
`docker image ls`


Запустить образ 
`docker run --rm -p 127.0.0.1:8000:8000 mycool:bomberman`

`docker run` – команда для запуска Docker-контейнера.
`rm` – параметр, который говорит Docker удалить контейнер после его завершения.
`-p 127.0.0.1:8000:8000` – параметр для проброса портов, он указывает Docker пробросить порт 8000 контейнера на localhost по IP-адресу 127.0.0.1.
`mycool:bomberman` – название образа, на основе которого будет создан и запущен контейнер. В нашем случае образ называется mycoolс тегом bomberman.


### Еще игрушки

```# pacman
--build-arg GAME_TAG=pacman
--build-arg GAME_URL=https://code.s3.yandex.net/Kubernetes/pacman.zip?etag=914e810db328253ff2db6073036ea032
--build-arg GAME_ARGS=PACMAN.exe

# bomberman
--build-arg GAME_TAG=bomberman
--build-arg GAME_URL=https://code.s3.yandex.net/Kubernetes/dyna.zip
--build-arg GAME_ARGS=dyna.exe

# keen
--build-arg GAME_TAG=keen
--build-arg GAME_URL=https://code.s3.yandex.net/Kubernetes/Keen.zip?etag=43ee6d837c7551ab400ef8ea2c19c4da
--build-arg GAME_ARGS=KEEN.BAT

# doom
--build-arg GAME_TAG=doom
--build-arg GAME_URL=https://code.s3.yandex.net/Kubernetes/Doom2.zip?etag=45dffe4dc2664b8c59302590cfc81a9b
--build-arg GAME_ARGS=DOOM.EXE

# scorch
--build-arg GAME_TAG=scorch
--build-arg GAME_URL=https://code.s3.yandex.net/Kubernetes/Scorcher.zip?etag=99ccc295530ac9608d7b9e9fac0e9d0d
--build-arg GAME_ARGS=SCORCH.EXE```