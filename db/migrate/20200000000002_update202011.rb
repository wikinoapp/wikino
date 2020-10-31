# frozen_string_literal: true

class Update202011 < ActiveRecord::Migration[6.0]
  def change
    add_column :notes, :modified_at, :datetime
  end
end
