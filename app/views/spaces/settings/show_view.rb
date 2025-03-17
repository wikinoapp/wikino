# typed: strict
# frozen_string_literal: true

module Spaces
  module Settings
    class ShowView < ApplicationView
      sig do
        params(
          current_user_entity: UserEntity,
          space_entity: SpaceEntity
        ).void
      end
      def initialize(current_user_entity:, space_entity:)
        @current_user_entity = current_user_entity
        @space_entity = space_entity
      end

      sig { override.void }
      def before_render
        title = I18n.t("meta.title.spaces.settings.show", space_name: space_entity.name)
        helpers.set_meta_tags(title:, **default_meta_tags)
      end

      sig { returns(UserEntity) }
      attr_reader :current_user_entity
      private :current_user_entity

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
