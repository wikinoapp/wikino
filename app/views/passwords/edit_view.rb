# typed: strict
# frozen_string_literal: true

module Passwords
  class EditView < ApplicationView
    sig { params(form: PasswordResetForm::Creation).void }
    def initialize(form:)
      @form = form
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.passwords.edit")
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(PasswordResetForm::Creation) }
    attr_reader :form
    private :form

    sig { returns(PageName) }
    private def current_page_name
      PageName::PasswordEdit
    end
  end
end
