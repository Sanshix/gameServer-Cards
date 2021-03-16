package base

import (
	"errors"
	"gameServer-demo/src/common"
	"strconv"

	pb "gameServer-demo/src/grpc"

	"github.com/golang/protobuf/proto"
	"github.com/streadway/amqp"
)

func init() {
	common.AllComponentMap["MQ"] = &MQ{}
}

// MQ 消息队列组件
type MQ struct {
	Base
	common.MQI
	MQConnection                    *amqp.Connection
	PlayerMsgSubChannel             *amqp.Channel
	PlayerMsgPublishChannel         *amqp.Channel
	ReportMsgPublishChannel         *amqp.Channel
	PlayerStartToPlayPublishChannel *amqp.Channel
}

// LoadComponent 加载消息队列组件
func (obj *MQ) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.Base.LoadComponent(config, componentName)
	err := obj.CheckConnection()
	if err != nil {
		panic(err)
	}
	common.LogInfo("mq load ok")
	return
}

// CheckConnection 初始化连接
func (obj *MQ) CheckConnection() error {
	if obj.MQConnection == nil || obj.MQConnection.IsClosed() == true {
		conn, err := amqp.Dial((*obj.Config)["connect_string"])
		if err != nil {
			common.LogError("MQ CheckConnection Dial has err", err)
			return err
		}
		obj.MQConnection = conn
		if obj.PlayerMsgSubChannel != nil {
			obj.PlayerMsgSubChannel.Close()
		}
		obj.PlayerMsgSubChannel = nil
		if obj.PlayerMsgPublishChannel != nil {
			obj.PlayerMsgPublishChannel.Close()
		}
		obj.PlayerMsgPublishChannel = nil
		if obj.ReportMsgPublishChannel != nil {
			obj.ReportMsgPublishChannel.Close()
		}
		if obj.PlayerStartToPlayPublishChannel != nil {
			obj.PlayerStartToPlayPublishChannel.Close()
		}
		obj.PlayerStartToPlayPublishChannel = nil
	}
	if obj.PlayerMsgSubChannel == nil {
		ch, err := obj.MQConnection.Channel()
		if err != nil {
			common.LogError("MQ CheckConnection PlayerMsgSubChannel has err", err)
			return err
		}
		obj.PlayerMsgSubChannel = ch
	}
	if obj.PlayerMsgPublishChannel == nil {
		ch, err := obj.MQConnection.Channel()
		if err != nil {
			common.LogError("MQ CheckConnection PlayerMsgPublishChannel has err", err)
			return err
		}
		obj.PlayerMsgPublishChannel = ch
	}
	if obj.ReportMsgPublishChannel == nil {
		ch, err := obj.MQConnection.Channel()
		if err != nil {
			common.LogError("MQ CheckConnection ReportMsgPublishChannel has err", err)
			return err
		}
		obj.ReportMsgPublishChannel = ch
	}

	if obj.PlayerStartToPlayPublishChannel == nil {
		ch, err := obj.MQConnection.Channel()
		if err != nil {
			common.LogError("MQ CheckConnection PlayerStartToPlayPublishChannel has err", err)
			return err
		}
		obj.PlayerStartToPlayPublishChannel = ch
	}

	return nil
}

// BindUser 绑定用户到队列
// 当用户登陆时绑定用户到队列
func (obj *MQ) BindUser(uuid string, subCallBack common.PlayerMsgSubFunc) error {
	err := obj.CheckConnection()
	if err != nil {
		common.LogError("MQ BindUser CheckConnection has err", err)
		return err
	}

	err = obj.PlayerMsgSubChannel.Cancel(uuid, false)
	if err != nil {
		common.LogError("MQ BindUser Cancel has err", err)
		return err
	}
	queueName := common.PlayerMsgQueue + uuid
	// name:队列名称;durable:是否持久化,队列存盘,true服务重启后信息不会丢失,影响性能;autoDelete:是否自动删除;noWait:是否非阻塞,
	// true为是,不等待RMQ返回信息;args:参数,传nil即可;exclusive:是否设置排他
	_, err = obj.PlayerMsgSubChannel.QueueDeclare(queueName, false, true, false, false, nil)
	if err != nil {
		common.LogError("MQ BindUser QueueDeclare has err", err)
		return err
	}
	err = obj.PlayerMsgSubChannel.Qos(1, 0, false)
	if err != nil {
		common.LogError("MQ BindUser Qos has err", err)
		return err
	}
	go func() {
		msgList, err := obj.PlayerMsgSubChannel.Consume(queueName, uuid, false, false, false, false, nil)
		if err != nil {
			common.LogError("MQ Consume Consume has err", err)
			return
		}
		for msg := range msgList {
			// 处理数据
			err := subCallBack(msg.Body)
			if err != nil {
				err = msg.Ack(true)
				if err != nil {
					common.LogError("MQ Consume Ack true has err", err)
					continue
				}
			} else {
				// 确认消息,必须为false
				err = msg.Ack(false)
				if err != nil {
					common.LogError("MQ Consume Ack false has err", err)
					continue
				}
			}
		}
	}()

	return nil
}

// UnBindUser 取消绑定用户到队列
// 当用户离线时取消绑定用户到队列
func (obj *MQ) UnBindUser(uuid string) error {
	err := obj.CheckConnection()
	if err != nil {
		common.LogError("MQ UnBindUser CheckConnection has err", err)
		return err
	}
	err = obj.PlayerMsgSubChannel.Cancel(uuid, false)
	if err != nil {
		common.LogError("MQ UnBindUser Cancel has err", err)
		return err
	}
	return nil
}

// SendToUser 发送消息到指定用户
func (obj *MQ) SendToUser(uuid string, msg []byte) error {
	queueName := common.PlayerMsgQueue + uuid
	/*err := obj.CheckConnection()
	if err != nil {
		common.LogError("MQ SendToUser CheckConnection has err", err)
		return err
	}*/
	// 发送任务消息
	err := obj.CheckConnection()
	if err != nil {
		common.LogError("MQ SendToUser CheckConnection has err", err)
		return err
	}
	err = obj.PlayerMsgPublishChannel.Publish("", queueName, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        msg,
	})
	if err != nil {
		common.LogError("MQ SendToUser Publish has err", err)
		return err
	}

	return nil
}

// SendReport 发送
func (obj *MQ) SendReport(msg []byte) error {
	if common.IsDev == true {
		return nil
	}
	queueName := common.ReportMsgQueue
	/*err := obj.CheckConnection()
	if err != nil {
		common.LogError("MQ SendToUser CheckConnection has err", err)
		return err
	}*/
	// 发送任务消息
	err := obj.CheckConnection()
	if err != nil {
		common.LogError("MQ SendReport CheckConnection has err", err)
		return err
	}
	err = obj.ReportMsgPublishChannel.Publish("", queueName, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        msg,
	})
	if err != nil {
		common.LogError("MQ SendReport Publish has err", err)
		return err
	}

	return nil
}

// BindReport 绑定报表消费者
func (obj *MQ) BindReport(uuid string, subCallBack common.ReportMsgSubFunc) error {
	if common.IsDev == true {
		return nil
	}
	err := obj.CheckConnection()
	if err != nil {
		common.LogError("MQ BindReport CheckConnection has err", err)
		return err
	}
	err = obj.ReportMsgPublishChannel.Cancel(uuid, false)
	if err != nil {
		common.LogError("MQ BindReport Cancel has err", err)
		return err
	}
	queueName := common.ReportMsgQueue
	// name:队列名称;durable:是否持久化,队列存盘,true服务重启后信息不会丢失,影响性能;autoDelete:是否自动删除;noWait:是否非阻塞,
	// true为是,不等待RMQ返回信息;args:参数,传nil即可;exclusive:是否设置排他
	_, err = obj.ReportMsgPublishChannel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		common.LogError("MQ BindReport QueueDeclare has err", err)
		return err
	}
	err = obj.ReportMsgPublishChannel.Qos(10, 0, true)
	if err != nil {
		common.LogError("MQ BindReport Qos has err", err)
		return err
	}
	go func() {
		msgList, err := obj.ReportMsgPublishChannel.Consume(queueName, uuid, false, false, false, false, nil)
		if err != nil {
			common.LogError("MQ Consume Consume has err", err)
			return
		}
		for msg := range msgList {
			// 处理数据
			err := subCallBack(msg.Body)
			if err != nil {
				// 重新入队，否则未确认的消息会持续占用内存
				err = msg.Reject(true)
				//err = msg.Ack(true)
				if err != nil {
					common.LogError("MQ Consume Ack true has err", err)
					continue
				}
			} else {
				// 确认消息,必须为false
				err = msg.Ack(false)
				if err != nil {
					common.LogError("MQ Consume Ack false has err", err)
					continue
				}
			}
		}
	}()

	return nil
}

// UnBindReport 取消绑定报表消费者
func (obj *MQ) UnBindReport(uuid string) error {
	err := obj.ReportMsgPublishChannel.Cancel(uuid, false)
	if err != nil {
		common.LogError("MQ UnBindReport Cancel has err", err)
		return err
	}
	return nil
}

//发送玩家开始玩游戏通知
func (obj *MQ) SendPlayerStartToPlay(msg *pb.PlayerStartPlayGameMessage) error {
	if common.IsDev == true {
		return nil
	}
	//queueName := common.PlayerStartToPlayQueue
	// 发送任务消息
	err := obj.CheckConnection()
	if err != nil {
		common.LogError("MQ SendPlayerStartToPlay CheckConnection has err", err)
		return err
	}
	//args := make(map[string]interface{})

	reportByte, err := proto.Marshal(msg)
	if err != nil {
		common.LogError("SendPlayerStartToPlay Drive Marshal msg has err", msg, err)
		return errors.New("SendPlayerStartToPlay Drive Marshal msg has err")
	}

	err = obj.PlayerStartToPlayPublishChannel.ExchangeDeclare(common.PlayerStartPlayGameExchange,
		"topic",
		true,  //是否持久化，RabbitMQ关闭后，没有持久化的Exchange将被清除
		false, //是否自动删除，如果没有与之绑定的Queue，直接删除
		false, //是否内置的，如果为true，只能通过Exchange到Exchange
		false,
		nil)
	if err != nil {
		common.LogError("MQ SendPlayerStartToPlay ExchangeDeclare has err", err)
		return err
	}

	key := common.PlayerStartPlayGameExchangeKey + strconv.Itoa(int(msg.GameType)) + "." + strconv.Itoa(int(msg.RoomType)) + "." + msg.RoomId + "." + msg.RoundId

	err = obj.PlayerStartToPlayPublishChannel.Publish(common.PlayerStartPlayGameExchange, key, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        reportByte,
	})
	if err != nil {
		common.LogError("MQ SendPlayerStartToPlay Publish has err", err)
		return err
	}

	return nil
}

func (obj *MQ) BindPlayerStartToPlay(uuid string, subCallback common.PlayerStartToPlayFunc) error {
	if common.IsDev == true {
		return nil
	}
	err := obj.CheckConnection()
	if err != nil {
		common.LogError("MQ BindPlayerStartToPlay CheckConnection has err", err)
		return err
	}

	err = obj.PlayerStartToPlayPublishChannel.ExchangeDeclare(common.PlayerStartPlayGameExchange,
		"topic",
		true,  //是否持久化，RabbitMQ关闭后，没有持久化的Exchange将被清除
		false, //是否自动删除，如果没有与之绑定的Queue，直接删除
		false, //是否内置的，如果为true，只能通过Exchange到Exchange
		false,
		nil)
	if err != nil {
		common.LogError("MQ BindPlayerStartToPlay ExchangeDeclare has err", err)
		return err
	}
	q, err := obj.PlayerStartToPlayPublishChannel.QueueDeclare(
		"", //随机生成队列名称
		true,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		common.LogError("MQ BindPlayerStartToPlay QueueDeclare has err", err)
		return err
	}

	key := common.PlayerStartPlayGameExchangeKey + "#"

	err = obj.PlayerStartToPlayPublishChannel.QueueBind(
		q.Name,
		key,
		common.PlayerStartPlayGameExchange,
		false,
		nil,
	)
	if err != nil {
		common.LogError("MQ BindPlayerStartToPlay QueueBind has err", err)
		return err
	}

	return nil
}
