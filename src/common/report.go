package common

import (
	pb "gameServer-demo/src/grpc"
	"time"

	"github.com/golang/protobuf/ptypes"
)

// 调用ReportPublish组件转发到MQ（只负责转发）
func sendMessage(reportMessage *pb.ReportMessage) *pb.ErrorMessage {
	replyMessage := &pb.EmptyMessage{}

	msgErr := Router.Call("ReportPublish", "SendReport", reportMessage, replyMessage, nil)
	if msgErr != nil {
		return msgErr
	}
	return nil
}

// PushGameRecord 推送游戏记录
func PushGameRecord(gameRecord *pb.GameRecordReport) *pb.ErrorMessage {
	reportMsg := &pb.ReportMessage{}

	reportMsg.ReportType = pb.ReportType_ReportType_GameRecord
	reportAny, err := ptypes.MarshalAny(gameRecord)
	if err != nil {
		LogError("report PushGameRecord GRPC HandleMessage MarshalAny(reply) has err", err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reportMsg.ReportContent = reportAny

	return sendMessage(reportMsg)

}

// PushBalanceChangeRecord 推送金额变动记录
func PushBalanceChangeRecord(balanceChangeRecord *pb.BalanceChangeRecordReport) *pb.ErrorMessage {
	reportMsg := &pb.ReportMessage{}

	reportMsg.ReportType = pb.ReportType_ReportType_BalanceChangeRecord
	reportAny, err := ptypes.MarshalAny(balanceChangeRecord)
	if err != nil {
		LogError("report PushBalanceChangeRecord GRPC HandleMessage MarshalAny(reply) has err", err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reportMsg.ReportContent = reportAny
	return sendMessage(reportMsg)
}

// PushPlayerBalanceChangeRecord 发送金额变动通知
func PushPlayerBalanceChangeRecord(playerInfo *pb.PlayerInfo, changeBalance int64, changeReason pb.ResourceChangeReason) *pb.ErrorMessage {
	//发送金额变动通知
	balanceChangeRecord := &pb.BalanceChangeRecordReport{}
	balanceChangeRecord.PlayerAccount = playerInfo.Account
	balanceChangeRecord.PlayerShortId = playerInfo.ShortId
	balanceChangeRecord.PlayerUuid = playerInfo.Uuid
	balanceChangeRecord.ChangeAmount = changeBalance
	balanceChangeRecord.BeforeBalance = playerInfo.Balance - changeBalance
	balanceChangeRecord.FinalBalance = playerInfo.Balance
	balanceChangeRecord.ChangeTime = time.Now().Unix()
	balanceChangeRecord.ChangeReason = changeReason
	return PushBalanceChangeRecord(balanceChangeRecord)
}
