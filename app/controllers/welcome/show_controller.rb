# typed: strict
# frozen_string_literal: true

class Welcome::ShowController < ApplicationController
  include Authenticatable

  sig { returns(T.untyped) }
  def call
    redirect_to(note_list_path) if user_signed_in?
  end
end
