# typed: strict
# frozen_string_literal: true

module EditSuggestions
  class CreateService < ApplicationService
    class Result < T::Struct
      const :edit_suggestion_record, EditSuggestionRecord
      const :edit_suggestion_page_record, EditSuggestionPageRecord
    end

    sig do
      params(
        space_member_record: SpaceMemberRecord,
        page_record: PageRecord,
        title: String,
        description: String,
        page_title: String,
        page_body: String
      ).returns(Result)
    end
    def call(space_member_record:, page_record:, title:, description:, page_title:, page_body:)
      topic_record = page_record.topic_record.not_nil!
      space_record = page_record.space_record.not_nil!

      # 編集提案のステータスは最初はdraft
      status = EditSuggestionStatus::Draft.serialize

      with_transaction do
        # 編集提案を作成
        edit_suggestion_record = EditSuggestionRecord.create!(
          space_record:,
          topic_record:,
          created_space_member_record: space_member_record,
          title:,
          description:,
          status:
        )

        # 編集提案ページを作成（最初はlatest_revision無し）
        edit_suggestion_page_record = EditSuggestionPageRecord.new(
          space_record:,
          edit_suggestion_record:,
          page_record:,
          page_revision_record: page_record&.revision_records&.order(created_at: :desc)&.first
        )
        edit_suggestion_page_record.save!(validate: false)

        # body_htmlを生成
        topic = TopicRepository.new.to_model(topic_record:)
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

        Result.new(
          edit_suggestion_record:,
          edit_suggestion_page_record:
        )
      end
    end
  end
end
