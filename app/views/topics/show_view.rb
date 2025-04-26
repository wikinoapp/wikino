# typed: strict
# frozen_string_literal: true

module Topics
  class ShowView < ApplicationView
    sig do
      params(
        current_user: T.nilable(User),
        topic: Topic,
        pinned_pages: T::Array[Page],
        page_list: PageList
      ).void
    end
    def initialize(current_user:, topic:, pinned_pages:, page_list:)
      @current_user = current_user
      @topic = topic
      @pinned_pages = pinned_pages
      @page_list = page_list
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.topics.show", space_name: space.name, topic_name: topic.name)
      helpers.set_meta_tags(title:, **default_meta_tags(site: false))
    end

    sig { returns(T.nilable(User)) }
    attr_reader :current_user
    private :current_user

    sig { returns(Topic) }
    attr_reader :topic
    private :topic

    sig { returns(T::Array[Page]) }
    attr_reader :pinned_pages
    private :pinned_pages

    sig { returns(PageList) }
    attr_reader :page_list
    private :page_list

    delegate :space, to: :topic
    delegate :pages, :pagination, to: :page_list

    sig { returns(T::Boolean) }
    private def signed_in?
      !current_user.nil?
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::TopicDetail
    end
  end
end
