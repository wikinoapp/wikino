# typed: strict
# frozen_string_literal: true

module Accounts
  class NewView < ApplicationView
    sig { params(form: AccountForm::Creation).void }
    def initialize(form:)
      @form = form
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.accounts.new")
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(AccountForm::Creation) }
    attr_reader :form
    private :form

    sig { returns(PageName) }
    private def current_page_name
      PageName::AccountNew
    end
  end
end
