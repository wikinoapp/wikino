# typed: strict
# frozen_string_literal: true

module Layouts
  class BasicComponent < ApplicationComponent
    renders_one :header
    renders_one :main
    renders_one :footer

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
  end
end
