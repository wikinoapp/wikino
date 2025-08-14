# typed: strict
# frozen_string_literal: true

module Spaces
  module Settings
    module Attachments
      class IndexView < ApplicationView
        sig do
          params(
            current_user: User,
            space: Space,
            attachments: T::Array[Attachment],
            total_count: Integer,
            current_page: Integer,
            per_page: Integer,
            search_query: T.nilable(String)
          ).void
        end
        def initialize(current_user:, space:, attachments:, total_count:, current_page:, per_page:, search_query:)
          @current_user = current_user
          @space = space
          @attachments = attachments
          @total_count = total_count
          @current_page = current_page
          @per_page = per_page
          @search_query = search_query
        end

        sig { override.void }
        def before_render
          title = I18n.t("meta.title.spaces.settings.attachments.index", space_name: space.name)
          helpers.set_meta_tags(title:, **default_meta_tags)
        end

        sig { returns(User) }
        attr_reader :current_user
        private :current_user

        sig { returns(Space) }
        attr_reader :space
        private :space

        sig { returns(T::Array[Attachment]) }
        attr_reader :attachments
        private :attachments

        sig { returns(Integer) }
        attr_reader :total_count
        private :total_count

        sig { returns(Integer) }
        attr_reader :current_page
        private :current_page

        sig { returns(Integer) }
        attr_reader :per_page
        private :per_page

        sig { returns(T.nilable(String)) }
        attr_reader :search_query
        private :search_query

        sig { returns(PageName) }
        private def current_page_name
          PageName::SpaceSettingsAttachments
        end

        sig { returns(Integer) }
        private def total_pages
          (total_count.to_f / per_page).ceil
        end

        sig { returns(T::Boolean) }
        private def has_previous_page?
          current_page > 1
        end

        sig { returns(T::Boolean) }
        private def has_next_page?
          current_page < total_pages
        end
      end
    end
  end
end
