# typed: true
# frozen_string_literal: true

module Pages
  class IndexController < ApplicationController
    #   include Authenticatable
    #
    #   before_action :authenticate_user
    #
    #   sig { returns(T.untyped) }
    #   def call
    #     @pages = T.must(current_user).pages.order(modified_at: :desc)
    #   end
  end
end
