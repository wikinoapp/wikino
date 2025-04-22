# typed: strict
# frozen_string_literal: true

module Spaces
  module Settings
    class ShowView < ApplicationView
      sig do
        params(
          current_user: User,
          space_entity: SpaceEntity
        ).void
      end
      def initialize(current_user:, space_entity:)
        @current_user = current_user
        @space_entity = space_entity
      end

      sig { override.void }
      def before_render
        title = I18n.t("meta.title.spaces.settings.show", space_name: space_entity.name)
        helpers.set_meta_tags(title:, **default_meta_tags)
      end

      sig { returns(User) }
      attr_reader :current_user
      private :current_user

      sig { returns(SpaceEntity) }
      attr_reader :space_entity
      private :space_entity

      sig { returns(PageName) }
      private def current_page_name
        PageName::SpaceSettings
      end
    end
  end
end
