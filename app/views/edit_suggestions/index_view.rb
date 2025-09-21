# typed: strict
# frozen_string_literal: true

module EditSuggestions
  class IndexView < ApplicationView
    sig do
      params(
        current_user: T.nilable(User),
        topic: Topic,
        edit_suggestions: T::Array[EditSuggestion],
        filter_state: String
      ).void
    end
    def initialize(current_user:, topic:, edit_suggestions:, filter_state:)
      @current_user = current_user
      @topic = topic
      @edit_suggestions = edit_suggestions
      @filter_state = filter_state
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.topics.edit_suggestions.index", space_name: space.name, topic_name: topic.name)
      helpers.set_meta_tags(title:, **default_meta_tags(site: false))
    end

    sig { returns(T.nilable(User)) }
    attr_reader :current_user
    private :current_user

    sig { returns(Topic) }
    attr_reader :topic
    private :topic

    sig { returns(T::Array[EditSuggestion]) }
    attr_reader :edit_suggestions
    private :edit_suggestions

    sig { returns(String) }
    attr_reader :filter_state
    private :filter_state

    delegate :space, to: :topic

    sig { returns(T::Boolean) }
    private def signed_in?
      !current_user.nil?
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::EditSuggestionList
    end

    sig { returns(T::Array[Topics::TabsComponent::TabItem]) }
    private def tabs
      [
        Topics::TabsComponent::TabItem.new(
          label: I18n.t("nouns.pages"),
          path: topic_path(space.identifier, topic.number),
          active: false
        ),
        Topics::TabsComponent::TabItem.new(
          label: I18n.t("nouns.edit_suggestions"),
          path: topic_edit_suggestion_list_path(space.identifier, topic.number),
          active: true
        )
      ]
    end

    sig { returns(T::Boolean) }
    private def is_open_filter?
      filter_state == "open"
    end

    sig { returns(T::Array[T::Hash[Symbol, T.untyped]]) }
    private def filter_tabs
      [
        {
          label: I18n.t("nouns.edit_suggestion_filter.open"),
          path: topic_edit_suggestion_list_path(space.identifier, topic.number, state: "open"),
          active: is_open_filter?,
          count: open_count
        },
        {
          label: I18n.t("nouns.edit_suggestion_filter.closed"),
          path: topic_edit_suggestion_list_path(space.identifier, topic.number, state: "closed"),
          active: !is_open_filter?,
          count: closed_count
        }
      ]
    end

    sig { returns(Integer) }
    private def open_count
      # TODO: 実際のカウントロジックは後で実装
      # EditSuggestionRecord.where(topic_record: topic_record).where(status: [draft, open]).count
      0
    end

    sig { returns(Integer) }
    private def closed_count
      # TODO: 実際のカウントロジックは後で実装
      # EditSuggestionRecord.where(topic_record: topic_record).where(status: [applied, closed]).count
      0
    end
  end
end
