# typed: strict
# frozen_string_literal: true

class EditSuggestionPageRecord < ApplicationRecord
  self.table_name = "edit_suggestion_pages"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :edit_suggestion_record, foreign_key: :edit_suggestion_id
  belongs_to :page_record, foreign_key: :page_id, optional: true
  belongs_to :page_revision_record, foreign_key: :page_revision_id, optional: true
  belongs_to :latest_revision_record, foreign_key: :latest_revision_id, class_name: "EditSuggestionPageRevisionRecord"

  has_many :revision_records, foreign_key: :edit_suggestion_page_id, class_name: "EditSuggestionPageRevisionRecord", dependent: :restrict_with_exception

  # 循環参照の問題を避けるため `edit_suggestion_pages.latest_revision_id` ではNULLを許容している
  # ただし、レコード更新時は必ずセットされて欲しいのでバリデーションでチェックする
  validates :latest_revision_id, presence: true, unless: :new_record?

  sig { returns(T::Boolean) }
  def new_page?
    page_id.nil? && page_revision_id.nil?
  end

  sig { returns(T::Boolean) }
  def existing_page?
    !new_page?
  end

  sig { returns(T::Boolean) }
  def title_changed?
    return true if new_page?

    page_revision_record.not_nil!.title != latest_revision_record.not_nil!.title
  end

  sig { returns(T::Boolean) }
  def body_changed?
    return true if new_page?

    page_revision_record.not_nil!.body != latest_revision_record.not_nil!.body
  end

  sig { returns(T::Boolean) }
  def has_changes?
    title_changed? || body_changed?
  end

  # 最新リビジョンの情報をプロキシするメソッド
  sig { returns(String) }
  def title
    latest_revision_record.not_nil!.title
  end

  sig { returns(String) }
  def body
    latest_revision_record.not_nil!.body
  end

  sig { returns(String) }
  def body_html
    latest_revision_record.not_nil!.body_html
  end

  # リビジョンを作成してHTMLを生成する
  sig do
    params(
      editor_space_member_record: SpaceMemberRecord,
      title: String,
      body: String
    ).returns(EditSuggestionPageRevisionRecord)
  end
  def create_revision_with_html!(editor_space_member_record:, title:, body:)
    # HTMLレンダリングに必要なModelオブジェクトを生成
    topic = TopicRepository.new.to_model(topic_record: edit_suggestion_record.not_nil!.topic_record.not_nil!)
    space = SpaceRepository.new.to_model(space_record: space_record.not_nil!)
    space_member = SpaceMemberRepository.new.to_model(space_member_record: editor_space_member_record)
    
    body_html = Markup.new(
      current_topic: topic,
      current_space: space,
      current_space_member: space_member
    ).render_html(text: body)

    revision = EditSuggestionPageRevisionRecord.create!(
      space_record:,
      edit_suggestion_page_record: self,
      editor_space_member_record:,
      title:,
      body:,
      body_html:
    )
    
    # latest_revisionを更新
    update!(latest_revision_record: revision)
    
    revision
  end
end
