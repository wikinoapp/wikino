# typed: strict
# frozen_string_literal: true

class EditSuggestionPageRevisionRecord < ApplicationRecord
  self.table_name = "edit_suggestion_page_revisions"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :edit_suggestion_page_record, foreign_key: :edit_suggestion_page_id
  belongs_to :editor_space_member_record, foreign_key: :editor_space_member_id, class_name: "SpaceMemberRecord"

  # リビジョンを作成してbody_htmlを生成する
  sig do
    params(
      space_record: SpaceRecord,
      edit_suggestion_page_record: EditSuggestionPageRecord,
      editor_space_member_record: SpaceMemberRecord,
      title: String,
      body: String,
      topic: Topic,
      space: Space,
      space_member: SpaceMember
    ).returns(EditSuggestionPageRevisionRecord)
  end
  def self.create_with_html!(space_record:, edit_suggestion_page_record:, editor_space_member_record:, title:, body:, topic:, space:, space_member:)
    body_html = Markup.new(
      current_topic: topic,
      current_space: space,
      current_space_member: space_member
    ).render_html(text: body)

    create!(
      space_record:,
      edit_suggestion_page_record:,
      editor_space_member_record:,
      title:,
      body:,
      body_html:
    )
  end
end
