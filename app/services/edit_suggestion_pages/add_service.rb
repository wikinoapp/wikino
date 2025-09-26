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
      with_transaction do
        edit_suggestion_page_record = edit_suggestion_record.add_or_update_page!(
          page_record:,
          space_member_record:,
          page_title:,
          page_body:
        )

        Result.new(edit_suggestion_page_record:)
      end
    end
  end
end
