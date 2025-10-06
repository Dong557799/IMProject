package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int
	//增加onlineMap表（在线用户表）以及channel管道
	OnlineMap map[string]*User
	mapLock   sync.RWMutex

	Message chan string
}

// sever接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

// 监听广播Message广播消息channel的goroutine,一旦有消息就发送给全部在线的User
func (this *Server) ListenMessager() {
	for {
		msg := <-this.Message
		//将message发送给全部在线的User
		this.mapLock.Lock()
		for _, cli := range this.OnlineMap {
			cli.C <- msg

		}
		this.mapLock.Unlock()
	}
}

// 广播消息的方法
func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg

	this.Message <- sendMsg
}
func (this *Server) Handler(conn net.Conn) {
	//fmt.Println("连接建立成功")
	user := NewUser(conn, this)
	user.Online()
	//监听用户是否活跃
	isLive := make(chan bool)
	//接受客户端发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Conn Read err:", err)
				return
			}
			//提取用户消息（去除"\n")
			msg := string(buf[:n-1])
			//用户针对msg进行消息处理
			user.DoMessage(msg)
			//用户的任意消息，代表当前用户活跃
			isLive <- true
		}
	}()
	//当前handler阻塞
	for {
		select {
		case <-isLive:
			//当前用户活跃，应该重置定时器
			//不做任何事情，为了激活select,更新下面的定时器
		case <-time.After(time.Second * 300):
			//已经超时
			//将当前User强制关闭
			user.SendMsg("你被踢了")
			//销毁资源
			close(user.C)
			//关闭链接
			//conn.Close()在ListenMessage中关闭chan不会增大cpu占有率
			//退出当前的handler
			return
		}
	}
}

// 启动服务器接口
func (this *Server) Start() {
	//socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	//close listen socket
	defer listener.Close()
	//启动监听msg的goroutine
	go this.ListenMessager()
	for {
		//accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept err:", err)
		}

		//do handler
		go this.Handler(conn)
	}

}
