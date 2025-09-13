# typed: strict
# frozen_string_literal: true

module Sidebar
  class ContentComponent < ApplicationComponent
    sig do
      params(
        current_page_name: PageName,
        current_user: T.nilable(User),
        current_space: T.nilable(Space)
      ).void
    end
    def initialize(current_page_name:, current_user:, current_space:)
      @current_page_name = current_page_name
      @current_user = current_user
      @current_space = current_space
    end

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name

    sig { returns(T.nilable(User)) }
    attr_reader :current_user
    private :current_user

    sig { returns(T.nilable(Space)) }
    attr_reader :current_space
    private :current_space

    sig { returns(T::Boolean) }
    private def signed_in?
      !current_user.nil?
    end

    sig { returns(String) }
    private def search_path_with_space_filter
      if current_space.present?
        search_path(q: "space:#{current_space.not_nil!.identifier}")
      else
        search_path
      end
    end
  end
end
