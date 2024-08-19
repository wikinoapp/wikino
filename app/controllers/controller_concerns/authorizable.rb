# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module Authorizable
    extend T::Sig
    extend ActiveSupport::Concern

    included do
      include Pundit::Authorization
    end

    sig { returns(User) }
    def pundit_user
      viewer!
    end
  end
end
