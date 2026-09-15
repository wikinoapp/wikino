package model

import "strconv"

// SpaceIDはスペースのID型
type SpaceID string

// TopicIDはトピックのID型
type TopicID string

// PageIDはページのID型
type PageID string

// SpaceMemberIDはスペースメンバーのID型
type SpaceMemberID string

// TopicMemberIDはトピックメンバーのID型
type TopicMemberID string

// DraftPageIDは下書きページのID型
type DraftPageID string

// PageRevisionIDはページリビジョンのID型
type PageRevisionID string

// DraftPageRevisionIDは下書きページリビジョンのID型
type DraftPageRevisionID string

// PageEditorIDはページ編集者のID型
type PageEditorID string

// PageAttachmentReferenceIDはページ添付ファイル参照のID型
type PageAttachmentReferenceID string

// UserIDはユーザーのID型
type UserID string

// AttachmentIDは添付ファイルのID型
type AttachmentID string

// SuggestionIDは編集提案のID型
type SuggestionID string

// SuggestionPageIDは編集提案ページのID型
type SuggestionPageID string

// SuggestionPageRevisionIDは編集提案ページリビジョンのID型
type SuggestionPageRevisionID string

// SuggestionCommentIDは編集提案コメントのID型
type SuggestionCommentID string

// FeatureFlagIDはフィーチャーフラグのID型
type FeatureFlagID string

// FeatureFlagNameはフィーチャーフラグ名の型
type FeatureFlagName string

// ExportIDはスペースのエクスポートのID型
type ExportID string

// SpaceIdentifierはスペース識別子の型
type SpaceIdentifier string

// PageNumberはページ番号の型
type PageNumber int32

// SuggestionNumberは編集提案番号の型
type SuggestionNumber int32

// StringはSpaceIDを文字列に変換する
func (id SpaceID) String() string { return string(id) }

// StringはTopicIDを文字列に変換する
func (id TopicID) String() string { return string(id) }

// StringはPageIDを文字列に変換する
func (id PageID) String() string { return string(id) }

// StringはSpaceMemberIDを文字列に変換する
func (id SpaceMemberID) String() string { return string(id) }

// StringはTopicMemberIDを文字列に変換する
func (id TopicMemberID) String() string { return string(id) }

// StringはDraftPageIDを文字列に変換する
func (id DraftPageID) String() string { return string(id) }

// StringはPageRevisionIDを文字列に変換する
func (id PageRevisionID) String() string { return string(id) }

// StringはDraftPageRevisionIDを文字列に変換する
func (id DraftPageRevisionID) String() string { return string(id) }

// StringはPageEditorIDを文字列に変換する
func (id PageEditorID) String() string { return string(id) }

// StringはPageAttachmentReferenceIDを文字列に変換する
func (id PageAttachmentReferenceID) String() string { return string(id) }

// StringはUserIDを文字列に変換する
func (id UserID) String() string { return string(id) }

// StringはAttachmentIDを文字列に変換する
func (id AttachmentID) String() string { return string(id) }

// StringはSuggestionIDを文字列に変換する
func (id SuggestionID) String() string { return string(id) }

// StringはSuggestionPageIDを文字列に変換する
func (id SuggestionPageID) String() string { return string(id) }

// StringはSuggestionPageRevisionIDを文字列に変換する
func (id SuggestionPageRevisionID) String() string { return string(id) }

// StringはSuggestionCommentIDを文字列に変換する
func (id SuggestionCommentID) String() string { return string(id) }

// StringはFeatureFlagIDを文字列に変換する
func (id FeatureFlagID) String() string { return string(id) }

// StringはFeatureFlagNameを文字列に変換する
func (n FeatureFlagName) String() string { return string(n) }

// StringはExportIDを文字列に変換する
func (id ExportID) String() string { return string(id) }

// StringはSpaceIdentifierを文字列に変換する
func (s SpaceIdentifier) String() string { return string(s) }

// StringはPageNumberを文字列に変換する
func (n PageNumber) String() string { return strconv.FormatInt(int64(n), 10) }

// SuggestionCommentNumberは編集提案コメント番号の型
type SuggestionCommentNumber int32

// StringはSuggestionNumberを文字列に変換する
func (n SuggestionNumber) String() string { return strconv.FormatInt(int64(n), 10) }

// StringはSuggestionCommentNumberを文字列に変換する
func (n SuggestionCommentNumber) String() string { return strconv.FormatInt(int64(n), 10) }

// TopicIDsToStringsはTopicIDスライスをstringスライスに変換する
func TopicIDsToStrings(ids []TopicID) []string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = string(id)
	}
	return s
}

// PageIDsToStringsはPageIDスライスをstringスライスに変換する
func PageIDsToStrings(ids []PageID) []string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = string(id)
	}
	return s
}

// SpaceMemberIDsToStringsはSpaceMemberIDスライスをstringスライスに変換する
func SpaceMemberIDsToStrings(ids []SpaceMemberID) []string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = string(id)
	}
	return s
}

// SpaceIDsToStringsはSpaceIDスライスをstringスライスに変換する。
func SpaceIDsToStrings(ids []SpaceID) []string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = string(id)
	}
	return s
}

// UserIDsToStringsはUserIDスライスをstringスライスに変換する
func UserIDsToStrings(ids []UserID) []string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = string(id)
	}
	return s
}

// StringsToPageIDsはstringスライスをPageIDスライスに変換する
func StringsToPageIDs(ss []string) []PageID {
	ids := make([]PageID, len(ss))
	for i, s := range ss {
		ids[i] = PageID(s)
	}
	return ids
}
