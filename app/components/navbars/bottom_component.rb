# typed: strict
# frozen_string_literal: true

module Navbars
  class BottomComponent < ApplicationComponent
    sig { params(current_page_name: PageName, current_user: T.nilable(User)).void }
    def initialize(current_page_name:, current_user:)
      @current_page_name = current_page_name
      @current_user = current_user
    end

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name

    sig { returns(T.nilable(User)) }
    attr_reader :current_user
    private :current_user

    sig { returns(T::Boolean) }
    private def signed_in?
      !current_user.nil?
    end
  end
end
