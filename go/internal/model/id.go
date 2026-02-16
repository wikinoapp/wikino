package model

// SpaceID はスペースのID型
type SpaceID string

// TopicID はトピックのID型
type TopicID string

// PageID はページのID型
type PageID string

// SpaceMemberID はスペースメンバーのID型
type SpaceMemberID string

// TopicMemberID はトピックメンバーのID型
type TopicMemberID string

// DraftPageID は下書きページのID型
type DraftPageID string

// String はSpaceIDを文字列に変換する
func (id SpaceID) String() string { return string(id) }

// String はTopicIDを文字列に変換する
func (id TopicID) String() string { return string(id) }

// String はPageIDを文字列に変換する
func (id PageID) String() string { return string(id) }

// String はSpaceMemberIDを文字列に変換する
func (id SpaceMemberID) String() string { return string(id) }

// String はTopicMemberIDを文字列に変換する
func (id TopicMemberID) String() string { return string(id) }

// String はDraftPageIDを文字列に変換する
func (id DraftPageID) String() string { return string(id) }

// PageIDsToStrings はPageIDスライスをstringスライスに変換する
func PageIDsToStrings(ids []PageID) []string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = string(id)
	}
	return s
}

// StringsToPageIDs はstringスライスをPageIDスライスに変換する
func StringsToPageIDs(ss []string) []PageID {
	ids := make([]PageID, len(ss))
	for i, s := range ss {
		ids[i] = PageID(s)
	}
	return ids
}
