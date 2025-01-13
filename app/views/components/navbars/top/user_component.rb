# typed: strict
# frozen_string_literal: true

module Navbars
  module Top
    class UserComponent < ApplicationComponent
      sig { params(current_user: User, class_name: String).void }
      def initialize(current_user:, class_name: "")
        @current_user = current_user
        @class_name = class_name
      end

      sig { returns(User) }
      attr_reader :current_user
      private :current_user

      sig { returns(String) }
      attr_reader :class_name
      private :class_name
    end
  end
end
