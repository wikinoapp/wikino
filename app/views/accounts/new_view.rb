# typed: strict
# frozen_string_literal: true

module Accounts
  class NewView < ApplicationView
    use_helpers :set_meta_tags

    sig { params(form: AccountForm).void }
    def initialize(form:)
      @form = form
    end

    sig { returns(AccountForm) }
    attr_reader :form
    private :form
  end
end
