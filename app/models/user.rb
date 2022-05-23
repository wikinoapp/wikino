# typed: strict
# frozen_string_literal: true

class User < ApplicationRecord
  include SoftDeletable

  has_many :notes, dependent: :destroy
end
