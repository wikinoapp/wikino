# typed: strict
# frozen_string_literal: true

module Navbars
  class TopComponent < ApplicationComponent
    sig { params(current_user: T.nilable(User), class_name: String).void }
    def initialize(current_user:, class_name: "")
      @current_user = current_user
      @class_name = class_name
    end

    sig { returns(T.nilable(User)) }
    attr_reader :current_user
    private :current_user

    sig { returns(String) }
    attr_reader :class_name
    private :class_name

    sig { returns(T::Boolean) }
    def signed_in?
      current_user.present?
    end
  end
end
