# typed: strict
# frozen_string_literal: true

module EditSuggestions
  class FormModalComponent < ApplicationComponent
    sig do
      params(
        space: Space,
        topic: Topic,
        page: T.nilable(Page),
        existing_edit_suggestions: T::Array[EditSuggestion]
      ).void
    end
    def initialize(space:, topic:, page:, existing_edit_suggestions:)
      @space = space
      @topic = topic
      @page = page
      @existing_edit_suggestions = existing_edit_suggestions
    end

    sig { returns(Space) }
    attr_reader :space
    private :space

    sig { returns(Topic) }
    attr_reader :topic
    private :topic

    sig { returns(T.nilable(Page)) }
    attr_reader :page
    private :page

    sig { returns(T::Array[EditSuggestion]) }
    attr_reader :existing_edit_suggestions
    private :existing_edit_suggestions

    sig { returns(T.nilable(String)) }
    private def new_edit_suggestion_form_path
      # ページが存在する場合のみパスを返す
      # 新規ページ作成の場合は、まだページが存在しないため、フォームは表示しない
      if page
        helpers.new_edit_suggestion_path(
          space_identifier: space.identifier,
          page_number: page.not_nil!.number
        )
      end
    end

    sig { returns(T.nilable(String)) }
    private def add_to_existing_path
      # ページが存在する場合のみパスを返す
      if page
        helpers.new_edit_suggestion_page_path(
          space_identifier: space.identifier,
          topic_number: topic.number,
          page_number: page.not_nil!.number
        )
      end
    end

    sig { returns(T::Boolean) }
    private def has_existing_edit_suggestions?
      existing_edit_suggestions.any?
    end
  end
end
