# typed: strict
# frozen_string_literal: true

module SignIn
  class ShowView < ApplicationView
    use_helpers :set_meta_tags

    sig { params(form: UserSessionForm).void }
    def initialize(form:)
      @current_page_name = PageName::SignIn
      @form = form
    end

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name

    sig { returns(UserSessionForm) }
    attr_reader :form
    private :form
  end
end
