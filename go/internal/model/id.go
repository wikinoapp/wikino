package model

import "strconv"

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

// PageRevisionID はページリビジョンのID型
type PageRevisionID string

// PageEditorID はページ編集者のID型
type PageEditorID string

// PageAttachmentReferenceID はページ添付ファイル参照のID型
type PageAttachmentReferenceID string

// UserID はユーザーのID型
type UserID string

// AttachmentID は添付ファイルのID型
type AttachmentID string

// SpaceIdentifier はスペース識別子の型
type SpaceIdentifier string

// PageNumber はページ番号の型
type PageNumber int32

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

// String はPageRevisionIDを文字列に変換する
func (id PageRevisionID) String() string { return string(id) }

// String はPageEditorIDを文字列に変換する
func (id PageEditorID) String() string { return string(id) }

// String はPageAttachmentReferenceIDを文字列に変換する
func (id PageAttachmentReferenceID) String() string { return string(id) }

// String はUserIDを文字列に変換する
func (id UserID) String() string { return string(id) }

// String はAttachmentIDを文字列に変換する
func (id AttachmentID) String() string { return string(id) }

// String はSpaceIdentifierを文字列に変換する
func (s SpaceIdentifier) String() string { return string(s) }

// String はPageNumberを文字列に変換する
func (n PageNumber) String() string { return strconv.FormatInt(int64(n), 10) }

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
