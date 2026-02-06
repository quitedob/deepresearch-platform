package service

import (
	"context"
	"log"
	"time"

	"github.com/ai-research-platform/internal/repository/dao"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	chatDAO       *dao.ChatDAO
	membershipDAO *dao.MembershipDAO
	userDAO       *dao.UserDAO
	stopChan      chan struct{}
	isRunning     bool
}

// NewScheduler 创建调度器
func NewScheduler(chatDAO *dao.ChatDAO, membershipDAO *dao.MembershipDAO, userDAO *dao.UserDAO) *Scheduler {
	return &Scheduler{
		chatDAO:       chatDAO,
		membershipDAO: membershipDAO,
		userDAO:       userDAO,
		stopChan:      make(chan struct{}),
	}
}

// Start 启动调度器
func (s *Scheduler) Start() {
	if s.isRunning {
		return
	}
	s.isRunning = true

	// 启动各个定时任务
	go s.runQuotaResetTask()
	go s.runAutoCleanTask()
	go s.runSessionCleanupTask()

	log.Println("[Scheduler] 定时任务调度器已启动")
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	if !s.isRunning {
		return
	}
	close(s.stopChan)
	s.isRunning = false
	log.Println("[Scheduler] 定时任务调度器已停止")
}

// runQuotaResetTask 配额重置任务
// 每小时检查需要重置配额的用户
func (s *Scheduler) runQuotaResetTask() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// 启动时立即执行一次
	s.resetExpiredQuotas()

	for {
		select {
		case <-ticker.C:
			s.resetExpiredQuotas()
		case <-s.stopChan:
			return
		}
	}
}

// resetExpiredQuotas 重置过期的配额
func (s *Scheduler) resetExpiredQuotas() {
	if s.membershipDAO == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	count, err := s.membershipDAO.ResetExpiredQuotas(ctx)
	if err != nil {
		log.Printf("[Scheduler] 重置配额失败: %v", err)
		return
	}

	if count > 0 {
		log.Printf("[Scheduler] 已重置 %d 个用户的配额", count)
	}
}

// runAutoCleanTask 自动清理任务
// 每天凌晨3点执行
func (s *Scheduler) runAutoCleanTask() {
	for {
		// 计算到下一个凌晨3点的时间
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		duration := next.Sub(now)

		select {
		case <-time.After(duration):
			s.cleanOldSessions()
		case <-s.stopChan:
			return
		}
	}
}

// cleanOldSessions 清理过期的会话
func (s *Scheduler) cleanOldSessions() {
	if s.chatDAO == nil || s.userDAO == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// 获取所有用户的自动清理设置
	users, err := s.userDAO.GetUsersWithAutoClean(ctx)
	if err != nil {
		log.Printf("[Scheduler] 获取用户自动清理设置失败: %v", err)
		return
	}

	totalCleaned := 0
	for _, user := range users {
		if user.AutoCleanDays <= 0 {
			continue
		}

		// 计算截止时间
		cutoffTime := time.Now().AddDate(0, 0, -user.AutoCleanDays)

		// 删除该用户的旧会话
		count, err := s.chatDAO.DeleteOldSessionsByUserID(ctx, user.ID, cutoffTime)
		if err != nil {
			log.Printf("[Scheduler] 清理用户 %s 的旧会话失败: %v", user.ID, err)
			continue
		}

		totalCleaned += count
	}

	if totalCleaned > 0 {
		log.Printf("[Scheduler] 自动清理完成，共删除 %d 个过期会话", totalCleaned)
	}
}

// runSessionCleanupTask 会话清理任务
// 每6小时清理一次空会话和孤立消息
func (s *Scheduler) runSessionCleanupTask() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupOrphanedData()
		case <-s.stopChan:
			return
		}
	}
}

// cleanupOrphanedData 清理孤立数据
func (s *Scheduler) cleanupOrphanedData() {
	if s.chatDAO == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 删除空会话（没有消息且创建超过24小时的会话）
	emptyCount, err := s.chatDAO.DeleteEmptySessions(ctx, 24*time.Hour)
	if err != nil {
		log.Printf("[Scheduler] 清理空会话失败: %v", err)
	} else if emptyCount > 0 {
		log.Printf("[Scheduler] 已清理 %d 个空会话", emptyCount)
	}

	// 删除孤立消息（会话已删除但消息还在）
	orphanCount, err := s.chatDAO.DeleteOrphanedMessages(ctx)
	if err != nil {
		log.Printf("[Scheduler] 清理孤立消息失败: %v", err)
	} else if orphanCount > 0 {
		log.Printf("[Scheduler] 已清理 %d 条孤立消息", orphanCount)
	}
}
