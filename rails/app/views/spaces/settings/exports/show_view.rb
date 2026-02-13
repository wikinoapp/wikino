# typed: strict
# frozen_string_literal: true

module Spaces
  module Settings
    module Exports
      class ShowView < ApplicationView
        sig do
          params(
            current_user: User,
            space: Space,
            export: Export,
            export_status: ExportStatus
          ).void
        end
        def initialize(current_user:, space:, export:, export_status:)
          @current_user = current_user
          @space = space
          @export = export
          @export_status = export_status
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

        sig { returns(Export) }
        attr_reader :export
        private :export

        sig { returns(ExportStatus) }
        attr_reader :export_status
        private :export_status

        sig { returns(String) }
        private def title
          I18n.t("meta.title.spaces.settings.exports.show", space_name: space.name)
        end

        sig { returns(PageName) }
        private def current_page_name
          PageName::SpaceSettingsExportDetail
        end
      end
    end
  end
end
