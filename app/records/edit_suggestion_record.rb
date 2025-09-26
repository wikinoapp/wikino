# typed: strict
# frozen_string_literal: true

class EditSuggestionRecord < ApplicationRecord
  self.table_name = "edit_suggestions"

  enum :status, {
    EditSuggestionStatus::Draft.serialize => 0,
    EditSuggestionStatus::Open.serialize => 1,
    EditSuggestionStatus::Applied.serialize => 2,
    EditSuggestionStatus::Closed.serialize => 3
  }, prefix: true

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :topic_record, foreign_key: :topic_id
  belongs_to :created_space_member_record, class_name: "SpaceMemberRecord", foreign_key: :created_space_member_id
  has_many :edit_suggestion_page_records, foreign_key: :edit_suggestion_id, dependent: :restrict_with_exception
  has_many :comment_records, class_name: "EditSuggestionCommentRecord", foreign_key: :edit_suggestion_id, dependent: :restrict_with_exception

  scope :by_status, ->(status) { where(status:) }
  scope :open_or_draft, -> { where(status: [EditSuggestionStatus::Draft.serialize, EditSuggestionStatus::Open.serialize]) }
  scope :closed_or_applied, -> { where(status: [EditSuggestionStatus::Closed.serialize, EditSuggestionStatus::Applied.serialize]) }

  sig { returns(T::Boolean) }
  def editable?
    status_draft? || status_open?
  end

  # 編集提案ページを作成する（最初はlatest_revisionなし）
  sig do
    params(
      page_record: T.nilable(PageRecord),
      page_revision_record: T.nilable(PageRevisionRecord)
    ).returns(EditSuggestionPageRecord)
  end
  def create_edit_suggestion_page_record!(page_record:, page_revision_record:)
    EditSuggestionPageRecord.new(
      space_record:,
      edit_suggestion_record: self,
      page_record:,
      page_revision_record:
    ).tap { |record| record.save!(validate: false) }
  end

  # 編集提案にページを追加または更新する
  sig do
    params(
      page_record: T.nilable(PageRecord),
      space_member_record: SpaceMemberRecord,
      page_title: String,
      page_body: String
    ).returns(EditSuggestionPageRecord)
  end
  def add_or_update_page!(page_record:, space_member_record:, page_title:, page_body:)
    # 既存の編集提案ページがある場合は取得、なければ作成
    edit_suggestion_page_record = edit_suggestion_page_records.find_by(page_record:)

    if edit_suggestion_page_record.nil?
      # 新規作成（最初はlatest_revision無し）
      edit_suggestion_page_record = create_edit_suggestion_page_record!(
        page_record:,
        page_revision_record: page_record&.revision_records&.order(created_at: :desc)&.first
      )
    end

    # リビジョンを作成（HTMLレンダリングも含む）
    edit_suggestion_page_record.create_revision_with_html!(
      editor_space_member_record: space_member_record,
      title: page_title,
      body: page_body
    )

    edit_suggestion_page_record
  end
end
