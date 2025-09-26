# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module EditSuggestionRenderable
    extend ActiveSupport::Concern
    extend T::Sig
    extend T::Helpers

    abstract!

    sig do
      params(
        form: EditSuggestionPages::CreateForm,
        page_record: PageRecord,
        topic_record: TopicRecord,
        space_member_record: T.nilable(SpaceMemberRecord)
      ).returns(T.untyped)
    end
    private def build_edit_suggestion_page_view(form:, page_record:, topic_record:, space_member_record:)
      page = PageRepository.new.to_model(page_record:, current_space_member: space_member_record)

      # 既存の編集提案を取得（フォームで選択可能な編集提案）
      existing_edit_suggestion_records = space_member_record
        .not_nil!
        .open_or_draft_edit_suggestion_records_for(topic_record:)
        .preload(:created_space_member_record)

      existing_edit_suggestions = EditSuggestionRepository.new.to_models(
        edit_suggestion_records: existing_edit_suggestion_records
      )

      EditSuggestionPages::NewView.new(form:, page:, existing_edit_suggestions:)
    end
  end
end
