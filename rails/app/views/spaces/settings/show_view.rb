# typed: strict
# frozen_string_literal: true

module Spaces
  module Settings
    class ShowView < ApplicationView
      sig { params(current_user: User, space: Space).void }
      def initialize(current_user:, space:)
        @current_user = current_user
        @space = space
      end

      sig { override.void }
      def before_render
        helpers.set_meta_tags(title:, **default_meta_tags)
      end

      sig { returns(User) }
      attr_reader :current_user
      private :current_user

      sig { returns(Space) }
      attr_reader :space
      private :space

      sig { returns(String) }
      private def title
        I18n.t("meta.title.spaces.settings.show", space_name: space.name)
      end

      sig { returns(PageName) }
      private def current_page_name
        PageName::SpaceSettings
      end
    end
  end
end
