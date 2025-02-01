# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module Authenticatable
    include ActionController::StrongParameters

    def self.helper_method(*args)
    end

    def cookies
    end

    def flash
    end

    def home_path
    end

    def home_url
    end

    def redirect_to(*args)
    end

    def request
    end

    def session
    end

    def sign_in_path
    end

    def space_path(*args)
    end

    def space_url(*args)
    end

    def t(*args)
    end
  end
end
