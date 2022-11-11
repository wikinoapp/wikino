# typed: strict
# frozen_string_literal: true

class Notes::IndexController < ApplicationController
  include Authenticatable

  before_action :authenticate_user

  sig { returns(T.untyped) }
  def call
    @notes = current_user.notes.order(modified_at: :desc)
  end
end
