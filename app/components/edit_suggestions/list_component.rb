# typed: strict
# frozen_string_literal: true

module EditSuggestions
  class ListComponent < ApplicationComponent
    sig { params(edit_suggestions: T::Array[EditSuggestion], topic: Topic).void }
    def initialize(edit_suggestions:, topic:)
      @edit_suggestions = edit_suggestions
      @topic = topic
    end

    sig { returns(T::Array[EditSuggestion]) }
    attr_reader :edit_suggestions
    private :edit_suggestions

    sig { returns(Topic) }
    attr_reader :topic
    private :topic

    delegate :space, to: :topic

    sig { params(edit_suggestion: EditSuggestion).returns(String) }
    private def status_label(edit_suggestion:)
      case edit_suggestion.status.serialize
      when EditSuggestionStatus::Draft.serialize
        I18n.t("nouns.edit_suggestion_status.draft")
      when EditSuggestionStatus::Open.serialize
        I18n.t("nouns.edit_suggestion_status.open")
      when EditSuggestionStatus::Applied.serialize
        I18n.t("nouns.edit_suggestion_status.applied")
      when EditSuggestionStatus::Closed.serialize
        I18n.t("nouns.edit_suggestion_status.closed")
      else
        ""
      end
    end

    sig { params(edit_suggestion: EditSuggestion).returns(String) }
    private def status_color_class(edit_suggestion:)
      case edit_suggestion.status.serialize
      when EditSuggestionStatus::Draft.serialize
        "bg-gray-100 text-gray-800"
      when EditSuggestionStatus::Open.serialize
        "bg-green-100 text-green-800"
      when EditSuggestionStatus::Applied.serialize
        "bg-purple-100 text-purple-800"
      when EditSuggestionStatus::Closed.serialize
        "bg-red-100 text-red-800"
      else
        "bg-gray-100 text-gray-800"
      end
    end

    sig { params(edit_suggestion: EditSuggestion).returns(String) }
    private def edit_suggestion_path(edit_suggestion:)
      # TODO: 編集提案詳細画面のルーティングが実装されたら更新
      "#"
    end
  end
end
