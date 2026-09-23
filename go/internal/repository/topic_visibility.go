package repository

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// TopicVisibilityはページ一覧を閲覧者が開けるトピックに絞り込む条件。
// トピックの集合はページ画面と同じ認可ルールで呼び出し元が解決する。
// これにより、判定がSQL上のスコープ名ではなくPolicy層に残る。
type TopicVisibility struct {
	// AllVisibleは絞り込み自体を行わないことを表す。
	AllVisible bool

	// TopicIDsは閲覧可能なトピックの一覧。AllVisibleがtrueのときは無視される。
	TopicIDs []model.TopicID
}

// AllTopicsVisibleは意図的にトピックの絞り込みを省略するTopicVisibilityを返す。
// この値自体は認可を付与・保証しない。現在は編集画面で全トピックのページタイトルを見せる
// 従来挙動を維持するために使用する。
func AllTopicsVisible() TopicVisibility {
	return TopicVisibility{AllVisible: true}
}

// VisibleTopicsは指定したトピックに限定したTopicVisibilityを返す。
func VisibleTopics(topicIDs []model.TopicID) TopicVisibility {
	return TopicVisibility{TopicIDs: topicIDs}
}
