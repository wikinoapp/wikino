# typed: strict
# frozen_string_literal: true

module Headers
  class GlobalComponent < ApplicationComponent
    renders_one :breadcrumbs

    sig { params(current_page_name: PageName, current_user_entity: T.nilable(UserEntity)).void }
    def initialize(current_page_name:, current_user_entity:)
      @current_page_name = current_page_name
      @current_user_entity = current_user_entity
    end

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name

    sig { returns(T.nilable(UserEntity)) }
    attr_reader :current_user_entity
    private :current_user_entity
  end
end
