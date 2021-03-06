godfs
==========
[![Build Status](https://travis-ci.org/hetianyi/godfs.svg?branch=master)](https://travis-ci.org/hetianyi/godfs)
[![go report card](https://goreportcard.com/badge/github.com/hetianyi/godfs "go report card")](https://goreportcard.com/report/github.com/hetianyi/godfs)

[README](README.md) | [中文文档](README_zh.md)
### ```godfs``` 是一个用go实现的轻量，快速，简单易用的分布式文件存储服务器。

```godfs``` 开箱即用，并对docker支持友好。

你可以在docker hub上下载最新镜像:
[https://hub.docker.com/r/hehety/godfs/](https://hub.docker.com/r/hehety/godfs/)

**[2019-03-28 更新] 最新版本1.1.1-dev因数据库改动不兼容之前版本!**

##### [2019-01-17 更新] 注意：最新的版本 1.1.0+ 和之前版本不兼容！

##### [2018-12-05 更新] Godfs 现已支持仪表盘来进行简单的资源监控了！

项目地址:[https://github.com/hetianyi/godfs-dashboard](https://github.com/hetianyi/godfs-dashboard)



![architecture](/doc/20180830151005.png)

## 特性

- 快速, 轻量, 开箱即用, 友好的API
- 易于扩展，运行稳定
- 非常低的资源开销
- 提供原生的客户端和Java客户端(还未开始)
- 提供HTTP方式的下载和上传API
- 支持文件断点下载
- 支持http上传和下载时的基本验证
- 跨站点资源保护
- 清晰的日志帮助排查运行异常
- 支持不同平台下的编译运行: Linux, Windows, Mac
- 更好地支持docker容器
- 文件分片保存
- 更好的文件迁移解决方案
- 支持读写和只读文件节点
- 文件组内自动同步
- 仪表盘支持(beta, 开发中)
- 支持访问令牌

## 安装

> 请先安装golang1.8+

以CentOS7为例.

### 从最新的源码构建：
```shell
yum install golang -y
git clone https://github.com/hetianyi/godfs.git
cd godfs
./make.sh
# Windows下直接点击 make.cmd 开始构建。
```
构建成功后, 三个文件会生成在`````./bin````` 目录下，分别是:
```shell
./bin/client
./bin/storage
./bin/tracker
```

将构建成功的二进制文件安装到目录 ```/usr/local/godfs```:
```shell
./install.sh /usr/local/godfs
```

启动tracker服务:
```shell
/usr/local/godfs/bin/tracker [-c /your/tracker/config/path]
```
启动storage服务:
```shell
/usr/local/godfs/bin/storage [-c /your/storage/config/path]
```
然后你就可以在命令行直接使用 ```client``` 来上传和下载文件了。
> 当然要先设置trackers服务器设置
```shell
# 例如，为客户端设置tracker服务器
client config set "trackers=host1:port1[,host2:port2]" "log_level=debug" ...
# config secret
client config set "secret=OASAD834jA97AAQE761=="
# 打印当前的配置
client config ls
```

举个栗子，上传一个文件:

```shell
 client upload [-g G01] /you/upload/file
```
![architecture](/doc/20180828095840.png)

如果你想上传文件到指定的group，可以在命令行加参数```-g <groupID>```

你还可以用一个更酷的命令来上传一个文件夹下所有的文件:
```shell
client upload *
```
![architecture](/doc/20180828100341.png)

如果你没有现成的godfs客户端，你可以使用 ```curl``` 来上传文件:
```shell
curl -F "file=@/your/file" "http://your.host:http_port/upload"
```
上传成功之后，服务器会返回一个json字符串:
```json
{
    "status":"success",
    "formData":{
        "data":[
            "G01/01/M/826d552525bceec5b8e9709efaf481ec"
        ],
        "name":[
            "mike"
        ]
    },
    "fileInfo":[
        {
            "index":0,
            "fileName":"mysql-cluster-community-7.6.7-1.sles12.x86_64.rpm-bundle.tar",
            "path":"G01/01/M/826d552525bceec5b8e9709efaf481ec"
        }
    ]
}
```

> 其中， ```formData``` 是post表单中的所有字段的name-value信息，文件已被替换为上传之后的路径地址(非英文字符将编码为十六进制字符串, 例如 '图片' --> '\\u56fe\\u7247')。
如果你想上传文件到指定的group，可以在路径上加参数```?group=<groupID>```

```shell
# 下载文件
client download G01/10/M/2c9da7ea280c020db7f4879f8180dfd6 --name 123.zip
```



#### Token的使用

token机制是参考FastDFS实现的，能够控制一个私有文件在一定时间内的可访问性。

token需要在后端自行生成，godfs只需要计算并匹配token，token携带的格式如下：

http://...?tk=<md5>&ts=<timestamp>

token计算：

md5(timestamp+filemd5+secret) ==> token



### 从最新源代码构建docker镜像：
```shell
cd godfs/docker
docker build -t godfs .
```
强烈推荐使用docker来运行godfs.
最新的godfs的docker镜像可以在 [docker hub](https://hub.docker.com/r/hehety/godfs/) 获取:
```shell
docker pull hehety/godfs
```

启动tracker服务器:
```shell
docker run -d -p 1022:1022 --name tracker --restart always -v /godfs/data:/godfs/data --privileged -e log_level="info" hehety/godfs:latest tracker
```

启动storage服务器:
```shell
docker run -d -p 1024:1024 -p 80:8001 --name storage -v /godfs/data:/godfs/data --privileged -e trackers=192.168.1.172:1022 -e advertise_addr=192.168.1.187 -e port=1024  -e instance_id="01" hehety/godfs storage
# 单机上部署多个storage最好加上命令： '-e port=1024'
```
这里，我们使用宿主机上的目录 ```/godfs/data``` 来存放上传的文件，你可以使用docker的命令```-e```来覆盖配置文件中的相应配置。

客户端命令:
```shell
NAME:
   godfs client cli
USAGE:
   client [global options] command [command options] [arguments...]
VERSION:
   1.1.0-beta
COMMANDS:
     upload    upload local files
     download  download a file
     inspect   inspect files information by md5
     config    client cli configuration settings operation
     help, h   Shows a list of commands or help for one command
GLOBAL OPTIONS:
   --trackers value               tracker servers (default: "127.0.0.1:1022")
   --log_level value              log level (trace, debug, info, warm, error, fatal) (default: "info")
   --log_rotation_interval value  log rotation interval h(hour),d(day),m(month),y(year) (default: "d")
   --secret value                 secret of trackers (trace, debug, info, warm, error, fatal)
   --help, -h                     show help
   --version, -v                  print the version
```



## 监控

godfs 仪表盘监控项目地址： [这里](https://github.com/hetianyi/godfs-dashboard)

这个项目目前处于开发中，能够监控godfs的一些基本状态信息。

可以通过docker达到开箱即用([这里](https://github.com/hetianyi/godfs-dashboard)).

```shell
# 开启监控
docker run -d -p 8080:80 --restart always --name godfs-dashboard hehety/godfs-dashboard
```

![architecture](/doc/20181205154643.png)

![architecture](/doc/20181205154909.png)



### 在工作站上做的基于tcp client的压力测试（1.1.0以及之后版本）

[测试方案](https://github.com/hetianyi/godfs/tree/master/example)

机器配置

| Node     | CPU                                      | CPU核心数 | 内存 | 磁盘 |
| -------- | ---------------------------------------- | :-------: | :--: | ---- |
| tracker1 | Intel(R) Xeon(R) CPU E5-1620 0 @ 3.60GHz |     1     | 1GB  | SSD  |
| tracker2 | Intel(R) Xeon(R) CPU E5-1620 0 @ 3.60GHz |     1     | 1GB  | SSD  |
| storage1 | Intel(R) Xeon(R) CPU E5-1620 0 @ 3.60GHz |     2     | 1GB  | SSD  |
| storage2 | Intel(R) Xeon(R) CPU E5-1620 0 @ 3.60GHz |     2     | 1GB  | SSD  |

测试汇总

| 参数           | 值                        |
| -------------- | ------------------------- |
| 类型           | 虚拟机                    |
| 操作系统       | CentOS7                   |
| tracker数      | 2                         |
| group数        | 2（G01,G02）              |
| storage数      | 4（G01x2,G02x2）          |
| docker         | 18.06.1-ce, build e68fc7a |
| 文件总数       | 5,000,000                 |
| 线程数         | 5                         |
| 总耗时         | 2h 35min                  |
| 系统平均吞吐量 | 537文件/秒                |
| 单节点吞吐量   | 134文件/秒                |
| 失败           | 0                         |

示意图

![architecture](doc/20190215153655.png)



### 在vultr上做的基于http client的压力测试（1.0.6以及之前版本）

|参数|值|
|---|---|
| OS        | CentOS7   |
| RAM       | 1GB       |
| CPU core  | 1         |
| DISK      | 60GB SSD  |

#### 测试说明
运行5个脚本分别产生100w个文件，这些文件内容是1~5000000的数字，生成的文件通过curl上传。


测试耗时41.26小时，并且没有出现失败，这意味着平均每秒有33.7个文件上传成功。

测试中主机的CPU使用率保持在60%-70%，tracker和storage消耗的内存均低于30M。
>注意：这里测试程序和单台的tracker，单台的storage运行在同一台主机上。
这个测试说明godfs在处理大并发（对于文件系统来说）的上传、数据库写入不成问题，对于稳定性来说也是一个很好的考验。

测试工具可以在release页面获取到。
### HTTP 下载测试

storage server 配置(California)

| Name     | Value   |
| -------- | ------- |
| OS       | CentOS7 |
| RAM      | 512M    |
| CPU core | 1       |
| DISK     | SSD     |

下载客户端机器配置(Los Angeles)

| Name     | Value   |
| -------- | ------- |
| OS       | CentOS7 |
| RAM      | 8GB     |
| CPU core | 4       |
| DISK     | SSD     |


#### 测试说明
这里使用 Apache [jmeter-5.0](http://jmeter.apache.org/download_jmeter.cgi) 作为测试工具l.

在测试中，我们使用20个线程下载4个不同大小的文件（文件大小不超过1MB），每个线程运行10000次，总共800000次。测试结果如下：

| Label | # Samples | Average | Median | 90% Line | 95% Line | 99% Line | Min  | Max  | Error % | Throughput | Received KB/sec | Sent KB/sec |
| ----- | --------- | ------- | ------ | -------- | -------- | -------- | ---- | ---- | ------- | ---------- | --------------- | ----------- |
| 1     | 200000    | 78      | 72     | 116      | 135      | 195      | 5    | 8377 | 0.00%   | 79.88966   | 20079.84        | 13.65       |
| 2     | 200000    | 39      | 36     | 66       | 76       | 106      | 2    | 2715 | 0.00%   | 79.89087   | 11202.43        | 13.65       |
| 3     | 200000    | 76      | 65     | 125      | 154      | 238      | 5    | 8641 | 0.00%   | 79.89052   | 37493.62        | 13.65       |
| 4     | 200000    | 55      | 49     | 91       | 111      | 171      | 4    | 2789 | 0.00%   | 79.891     | 23045.82        | 13.65       |
| Total | 800000    | 62      | 55     | 104      | 126      | 193      | 2    | 8641 | 0.00%   | 319.55492  | 91819.69        | 54.61       |

![architecture](/doc/response-time.png)

**测试结果**

| 样本总数     | 800000      |
| ------------ | ----------- |
| 线程         | 20          |
| 总耗时       | 41min       |
| 平均请求     | 319.55492/s |
| 平均响应时间 | 62ms        |
| 成功率       | 100%        |
| 失败率       | 0%          |

我将来会在这里发布更多的测试。



## 更新日志

2019/03/28
1. 参考FastDFS实现了访问令牌机制。

2019/01/17

1. 为了更好地开发，引入许多开源组件:

   [github.com/mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)

   [github.com/jinzhu/gorm](github.com/jinzhu/gorm)

   [github.com/json-iterator/go](github.com/json-iterator/go)

   [github.com/urfave/cli](https://github.com/urfave/cli)

2. 重写了底层的TCP通讯协议，让程序的可拓展性更强。

3. 代码重构

4. 重新设计了client端命令，使之更加规范合理。