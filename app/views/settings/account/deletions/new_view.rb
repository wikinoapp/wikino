# typed: strict
# frozen_string_literal: true

module Settings
  module Account
    module Deletions
      class NewView < ApplicationView
        sig do
          params(
            current_user: User,
            form: AccountForm::DestroyConfirmation,
            active_spaces: T::Array[Space]
          ).void
        end
        def initialize(current_user:, form:, active_spaces:)
          @current_user = current_user
          @form = form
          @active_spaces = active_spaces
        end

        sig { override.void }
        def before_render
          helpers.set_meta_tags(title: current_page_title, **default_meta_tags(site: false))
        end

        sig { returns(User) }
        attr_reader :current_user
        private :current_user

        sig { returns(AccountForm::DestroyConfirmation) }
        attr_reader :form
        private :form

        sig { returns(T::Array[Space]) }
        attr_reader :active_spaces
        private :active_spaces

        sig { returns(T::Boolean) }
        def can_destroy_account?
          active_spaces.size.zero?
        end

        sig { returns(String) }
        private def current_page_title
          I18n.t("meta.title.settings.account.deletions.new")
        end

        sig { returns(PageName) }
        private def current_page_name
          PageName::Settings
        end
      end
    end
  end
end
