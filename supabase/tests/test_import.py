import importlib.util
import json
import tempfile
import unittest
from pathlib import Path


script = Path(__file__).resolve().parents[1] / 'scripts/import_library.py'
spec = importlib.util.spec_from_file_location('import_library', script)
importer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(importer)

OWNER = '11541e19-44a5-4be8-9d18-0323bb725219'


class ImportTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.path = Path(self.directory.name) / 'export.json'

        common = {
            'id': 1,
            'cover_url': None,
            'spotify_id': None,
            'created_at': '2026-10-20T00:00:00Z',
        }

        self.export = {
            'format_version': 2,
            'application': 'better-music',
            'library': {
                'artists': [{**common, 'name': 'Artist'}],
                'albums': [
                    {
                        **common,
                        'artist_id': 1,
                        'title': "Quote ' and $$",
                        'year': None,
                        'listened': True,
                        'rating': 1,
                        'comment': None,
                        'listened_at': None,
                    },
                ],
            },
        }

    def load(self):
        self.path.write_text(json.dumps(self.export))
        return importer.load_export(self.path, OWNER)

    def test_ids_links_and_nullable_values_are_preserved(self):
        records, report = self.load()
        repeated, _ = self.load()

        self.assertEqual(records, repeated)
        self.assertEqual(
            records['albums'][0]['artist_id'],
            records['artists'][0]['id'],
        )
        self.assertIsNone(records['albums'][0]['comment'])
        self.assertEqual(report['albums'], 1)

    def test_invalid_ratings_links_and_fields_are_rejected(self):
        album = self.export['library']['albums'][0]

        for rating in [0, 11, True, 2.5]:
            with self.subTest(rating=rating):
                album['rating'] = rating
                with self.assertRaises(ValueError):
                    self.load()

        album['rating'] = 1
        album['artist_id'] = 99

        with self.assertRaisesRegex(ValueError, 'artist reference'):
            self.load()

        album['artist_id'] = 1
        album['user_id'] = OWNER

        with self.assertRaisesRegex(ValueError, 'fields'):
            self.load()

    def test_modes_share_checks_but_have_different_endings(self):
        records, report = self.load()
        rehearsal = importer.make_sql(records, report)
        final = importer.make_sql(records, report, save=True)

        self.assertIn('\nrollback;', rehearsal)
        self.assertNotIn('\ncommit;', rehearsal)
        self.assertIn('\ncommit;', final)
        self.assertNotIn('\nrollback;', final)

        for sql in [rehearsal, final]:
            self.assertIn('Target library is not empty', sql)
            self.assertIn('is distinct from', sql)
            self.assertIn("Quote ' and $$", sql)

    def test_empty_library_and_duplicate_names(self):
        self.export['library']['artists'].append(
            {**self.export['library']['artists'][0], 'id': 2, 'name': 'ARTIST'},
        )

        with self.assertRaisesRegex(ValueError, 'duplicate'):
            self.load()

        self.export['library'] = {'artists': [], 'albums': []}
        records, report = self.load()

        self.assertEqual(report['albums'], 0)
        self.assertIn('jsonb_populate_recordset', importer.make_sql(records, report))


if __name__ == '__main__':
    unittest.main()
