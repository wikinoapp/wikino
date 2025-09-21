# typed: strict
# frozen_string_literal: true

module EditSuggestionPages
  class AddService < ApplicationService
    class Result < T::Struct
      const :edit_suggestion_page_record, EditSuggestionPageRecord
    end

    sig do
      params(
        edit_suggestion_record: EditSuggestionRecord,
        space_member_record: SpaceMemberRecord,
        page_record: T.nilable(PageRecord),
        page_title: String,
        page_body: String
      ).returns(Result)
    end
    def call(edit_suggestion_record:, space_member_record:, page_record:, page_title:, page_body:)
      space_record = edit_suggestion_record.space_record.not_nil!
      topic_record = edit_suggestion_record.topic_record.not_nil!

      with_transaction do
        # 既存の編集提案ページがある場合は更新
        edit_suggestion_page_record = edit_suggestion_record.edit_suggestion_page_records.find_by(page_record:)

        if edit_suggestion_page_record.nil?
          # 新規作成（最初はlatest_revision無し）
          edit_suggestion_page_record = EditSuggestionPageRecord.new(
            space_record:,
            edit_suggestion_record:,
            page_record:,
            page_revision_record: page_record&.revision_records&.order(created_at: :desc)&.first
          )
          edit_suggestion_page_record.save!(validate: false)
        end

        # body_htmlを生成
        topic = TopicRepository.new.to_model(topic_record: topic_record.not_nil!)
        space = SpaceRepository.new.to_model(space_record: space_record.not_nil!)
        space_member = SpaceMemberRepository.new.to_model(space_member_record:)

        body_html = Markup.new(
          current_topic: topic,
          current_space: space,
          current_space_member: space_member
        ).render_html(text: page_body)

        # 編集提案ページリビジョンを作成
        edit_suggestion_page_revision_record = EditSuggestionPageRevisionRecord.create!(
          space_record:,
          edit_suggestion_page_record:,
          editor_space_member_record: space_member_record,
          title: page_title,
          body: page_body,
          body_html:
        )

        # latest_revision_idを更新
        edit_suggestion_page_record.update!(latest_revision_record: edit_suggestion_page_revision_record)

        Result.new(edit_suggestion_page_record:)
      end
    end
  end
end
