# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Deletions
      class NewView < ApplicationView
        sig { params(current_user: User, space: Space, form: Spaces::DestroyConfirmationForm).void }
        def initialize(current_user:, space:, form:)
          @current_user = current_user
          @space = space
          @form = form
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

        sig { returns(Spaces::DestroyConfirmationForm) }
        attr_reader :form
        private :form

        sig { returns(String) }
        private def title
          I18n.t("meta.title.spaces.settings.deletions.new", space_name: space.name)
        end

        sig { returns(PageName) }
        private def current_page_name
          PageName::SpaceSettingsDeletion
        end
      end
    end
  end
end
