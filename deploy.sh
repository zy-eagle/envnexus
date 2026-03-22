#!/bin/bash

# EnvNexus 管理脚本
# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 获取部署目录
DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/deploy/docker" && pwd)"

# 检查环境依赖
check_env() {
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}[错误] 未检测到 Docker，请先安装 Docker。${NC}"
        exit 1
    fi
    if ! docker compose version &> /dev/null; then
        echo -e "${RED}[错误] 未检测到 Docker Compose (V2)，请先安装或升级 Docker。${NC}"
        exit 1
    fi
}

# 打印访问地址
print_urls() {
    SERVER_IP=$(hostname -I | awk '{print $1}')
    if [ -z "$SERVER_IP" ]; then
        SERVER_IP="127.0.0.1"
    fi
    
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}          服务已启动！🎉                ${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo -e "你可以通过以下地址访问服务："
    echo -e "👉 ${YELLOW}控制台前端 (Console Web):${NC} http://${SERVER_IP}:3000"
    echo -e "👉 ${YELLOW}平台 API (Platform API):${NC}  http://${SERVER_IP}:8080"
    echo -e "👉 ${YELLOW}会话网关 (Session Gateway):${NC} http://${SERVER_IP}:8081"
    echo -e ""
    echo -e "查看服务运行状态: ${YELLOW}cd deploy/docker && docker compose ps${NC}"
    echo -e "查看服务日志:     ${YELLOW}cd deploy/docker && docker compose logs -f${NC}"
}

# 部署/启动服务
deploy() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}      EnvNexus 开始部署/启动所有服务    ${NC}"
    echo -e "${GREEN}========================================${NC}"
    
    cd "$DEPLOY_DIR" || exit 1

    # 准备环境变量文件
    if [ ! -f ".env" ]; then
        if [ -f ".env.example" ]; then
            echo -e "${YELLOW}[提示] 未发现 .env 文件，正在从 .env.example 复制...${NC}"
            cp .env.example .env
        else
            echo -e "${RED}[错误] 找不到 .env.example 文件，无法初始化配置。${NC}"
            exit 1
        fi
    fi

    # 确保数据持久化目录不会被 Git 追踪
    GITIGNORE_FILE="../../.gitignore"
    if [ -f "$GITIGNORE_FILE" ]; then
        if ! grep -q "deploy/docker/volumes/" "$GITIGNORE_FILE"; then
            echo "deploy/docker/volumes/" >> "$GITIGNORE_FILE"
            echo -e "${YELLOW}[提示] 已将 volumes 目录添加到 .gitignore 防止数据泄露。${NC}"
        fi
    fi

    echo -e "${GREEN}[执行] docker compose up -d --build...${NC}"
    docker compose up -d --build

    if [ $? -eq 0 ]; then
        print_urls
    else
        echo -e "${RED}[错误] 启动失败，请检查上方日志。${NC}"
    fi
}

# 仅部署前端
deploy_web() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}      EnvNexus 仅重新构建/部署前端      ${NC}"
    echo -e "${GREEN}========================================${NC}"
    
    cd "$DEPLOY_DIR" || exit 1
    echo -e "${GREEN}[执行] docker compose up -d --build console-web...${NC}"
    docker compose up -d --build console-web

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}[成功] 前端部署完成。${NC}"
    else
        echo -e "${RED}[错误] 前端部署失败。${NC}"
    fi
}

# 仅部署后端API
deploy_api() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}      EnvNexus 仅重新构建/部署后端API   ${NC}"
    echo -e "${GREEN}========================================${NC}"
    
    cd "$DEPLOY_DIR" || exit 1
    echo -e "${GREEN}[执行] docker compose up -d --build platform-api session-gateway job-runner...${NC}"
    docker compose up -d --build platform-api session-gateway job-runner

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}[成功] 后端部署完成。${NC}"
    else
        echo -e "${RED}[错误] 后端部署失败。${NC}"
    fi
}

# 关闭服务
stop() {
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}      EnvNexus 正在关闭所有服务         ${NC}"
    echo -e "${YELLOW}========================================${NC}"
    
    cd "$DEPLOY_DIR" || exit 1
    docker compose down
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}[成功] 所有服务已关闭。你的数据依然安全保存在 volumes 目录中。${NC}"
    else
        echo -e "${RED}[错误] 关闭服务时发生错误。${NC}"
    fi
}

# 重启服务
restart() {
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}      EnvNexus 正在重启所有服务         ${NC}"
    echo -e "${YELLOW}========================================${NC}"
    
    cd "$DEPLOY_DIR" || exit 1
    docker compose restart
    
    if [ $? -eq 0 ]; then
        print_urls
    else
        echo -e "${RED}[错误] 重启服务时发生错误。${NC}"
    fi
}

# 查看状态
status() {
    cd "$DEPLOY_DIR" || exit 1
    docker compose ps
}

# 主逻辑
check_env

case "$1" in
    start|deploy)
        deploy
        ;;
    web)
        deploy_web
        ;;
    api)
        deploy_api
        ;;
    stop|down)
        stop
        ;;
    restart)
        restart
        ;;
    status|ps)
        status
        ;;
    *)
        echo "使用方法: $0 {start|web|api|stop|restart|status}"
        echo ""
        echo "命令说明:"
        echo "  start   (或 deploy) : 部署并启动所有服务"
        echo "  web                 : 仅重新构建并部署前端 (console-web)"
        echo "  api                 : 仅重新构建并部署后端 (platform-api, gateway, runner)"
        echo "  stop    (或 down)   : 关闭并移除所有服务容器（不删除数据）"
        echo "  restart             : 重启所有服务（不重新构建）"
        echo "  status  (或 ps)     : 查看所有服务的运行状态"
        exit 1
        ;;
esac
