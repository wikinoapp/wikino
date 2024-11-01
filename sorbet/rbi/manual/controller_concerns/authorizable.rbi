# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module Authorizable
    include Pundit::Authorization

    def params
    end
  end
end
