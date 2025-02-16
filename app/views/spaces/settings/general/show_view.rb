# typed: strict
# frozen_string_literal: true

module Spaces
  module Settings
    module General
      class ShowView < ApplicationView
        sig { params(space_entity: SpaceEntity, form: EditSpaceForm).void }
        def initialize(space_entity:, form:)
          @space_entity = space_entity
          @form = form
        end

        def before_render
          title = I18n.t("meta.title.spaces.settings.general.show", space_name: space_entity.name)
          helpers.set_meta_tags(title:, **default_meta_tags)
        end

        sig { returns(SpaceEntity) }
        attr_reader :space_entity
        private :space_entity

        sig { returns(EditSpaceForm) }
        attr_reader :form
        private :form

        sig { returns(PageName) }
        private def current_page_name
          PageName::SpaceSettingsGeneral
        end
      end
    end
  end
end
