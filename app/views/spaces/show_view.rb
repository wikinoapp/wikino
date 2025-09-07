# typed: strict
# frozen_string_literal: true

module Spaces
  class ShowView < ApplicationView
    sig do
      params(
        current_user: T.nilable(User),
        joined_space: T::Boolean,
        space: Space,
        first_joined_topic: T.nilable(Topic),
        pinned_pages: T::Array[Page],
        page_list: PageList,
        topics: T::Array[Topic]
      ).void
    end
    def initialize(current_user:, joined_space:, space:, first_joined_topic:, pinned_pages:, page_list:, topics:)
      @current_user = current_user
      @joined_space = joined_space
      @space = space
      @first_joined_topic = first_joined_topic
      @pinned_pages = pinned_pages
      @page_list = page_list
      @topics = topics
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.spaces.show", space_name: space.name)
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(T.nilable(User)) }
    attr_reader :current_user
    private :current_user

    sig { returns(T::Boolean) }
    attr_reader :joined_space
    private :joined_space
    alias_method :joined_space?, :joined_space

    sig { returns(Space) }
    attr_reader :space
    private :space

    sig { returns(T.nilable(Topic)) }
    attr_reader :first_joined_topic
    private :first_joined_topic

    sig { returns(T::Array[Page]) }
    attr_reader :pinned_pages
    private :pinned_pages

    sig { returns(PageList) }
    attr_reader :page_list
    private :page_list

    sig { returns(T::Array[Topic]) }
    attr_reader :topics
    private :topics

    delegate :pages, :pagination, to: :page_list

    sig { returns(T::Boolean) }
    private def signed_in?
      !current_user.nil?
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::SpaceDetail
    end
  end
end
