# 表示依赖 alpine 最新版
FROM alpine:latest
MAINTAINER Peaceflash<peaceflash@gmail.com>
ENV VERSION 2.0

# 在容器根目录 创建一个 apps 目录
WORKDIR /apps
COPY ./go-ip2region /apps/golang_app
COPY ./ip2region.db /apps/ip2region.db
# 设置时区为上海
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN echo 'Asia/Shanghai' >/etc/timezone

# 设置编码
ENV LANG C.UTF-8
# 暴露端口
EXPOSE 9090
RUN ["chmod", "+x", "/apps/golang_app"]
# 运行golang程序的命令
ENTRYPOINT ["/apps/golang_app"]
