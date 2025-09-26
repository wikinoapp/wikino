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

      with_transaction do
        edit_suggestion_record = space_member_record.create_draft_edit_suggestion_record!(
          topic_record:,
          title:,
          description:
        )

        edit_suggestion_page_record = edit_suggestion_record.create_edit_suggestion_page_record!(
          page_record:,
          page_revision_record: page_record.latest_revision_record
        )

        edit_suggestion_page_record.create_revision_with_html!(
          editor_space_member_record: space_member_record,
          title: page_title,
          body: page_body
        )

        Result.new(
          edit_suggestion_record:,
          edit_suggestion_page_record:
        )
      end
    end
  end
end
