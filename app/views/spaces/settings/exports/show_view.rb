# typed: strict
# frozen_string_literal: true

module Spaces
  module Settings
    module Exports
      class ShowView < ApplicationView
        sig do
          params(
            current_user_entity: UserEntity,
            space_entity: SpaceEntity,
            export_entity: ExportEntity
          ).void
        end
        def initialize(current_user_entity:, space_entity:, export_entity:)
          @current_user_entity = current_user_entity
          @space_entity = space_entity
          @export_entity = export_entity
        end

        sig { override.void }
        def before_render
          title = I18n.t("meta.title.spaces.settings.exports.show", space_name: space_entity.name)
          helpers.set_meta_tags(title:, **default_meta_tags(site: false))
        end

        sig { returns(UserEntity) }
        attr_reader :current_user_entity
        private :current_user_entity

        sig { returns(SpaceEntity) }
        attr_reader :space_entity
        private :space_entity

        sig { returns(ExportEntity) }
        attr_reader :export_entity
        private :export_entity

        sig { returns(PageName) }
        private def current_page_name
          PageName::SpaceSettingsExportDetail
        end
      end
    end
  end
end
