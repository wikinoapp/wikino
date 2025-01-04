# typed: strict
# frozen_string_literal: true

module Views
  module Accounts
    class New < Views::Base
      sig { params(form: AccountForm).void }
      def initialize(form:)
        @form = form
      end

      sig { returns(AccountForm) }
      attr_reader :form
      private :form
    end
  end
end
